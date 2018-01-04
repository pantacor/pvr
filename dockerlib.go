package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema2"

	"github.com/jessfraz/reg/registry"
	"github.com/jessfraz/reg/utils"
	digest "github.com/opencontainers/go-digest"
	pb "gopkg.in/cheggaaa/pb.v1"

	"errors"
	"io/ioutil"
)

type DockerRegistry struct {
	pvr         *Pvr
	registry    *registry.Registry
	arch        string
	initialized bool
}

func (p *Pvr) NewDocker(reg string, arch string, username string, password string) (*DockerRegistry, error) {

	d := &DockerRegistry{
		pvr:  p,
		arch: arch,
	}

	if d.arch == "" {
		d.arch = runtime.GOARCH
	}

	if reg == "" {
		reg = "docker.io"
	}

	authConfig, err := utils.GetAuthConfig(username, password, reg)

	if err != nil {
		return nil, err
	}

	d.registry, err = registry.New(authConfig, false)

	if err != nil {
		return nil, err
	}

	d.initialized = true
	return d, nil
}

func splitNamedRef(namedRef string) (repo string, tag string) {
	i := strings.Index(namedRef, ":")

	if i < 0 {
		return namedRef, "latest"
	}

	return namedRef[:i], namedRef[i+1:]
}

type layerDownload struct {
	repo       string
	digest     digest.Digest
	size       int64
	targetPath string
	bar        *pb.ProgressBar
	err        error
}

func (d *DockerRegistry) layerDownloadWorker(jobs chan layerDownload, done chan layerDownload) {

	for v := range jobs {
		v.bar.Total = v.size
		v.bar.Units = pb.U_BYTES
		v.bar.UnitsWidth = 25
		v.bar.ShowSpeed = true

		reader, err := d.registry.DownloadLayer(v.repo, v.digest)
		if err != nil {
			v.err = err
			done <- v
			continue
		}

		fd, err := os.OpenFile(v.targetPath, os.O_CREATE, 0644)
		if err != nil {
			v.err = err
			done <- v
			continue
		}

		defer fd.Close()

		for {
			buf := make([]byte, 1024*64)

			n, err := reader.Read(buf)
			if n > 0 {
				_, err := fd.Write(buf[:n])
				if err != nil {
					v.err = err
					break
				}
				v.bar.Add(n)
			}

			if err != nil {
				v.err = err
				break
			}
		}

		done <- v
	}

}

type pgReader struct {
	io.Reader
	pgbar *pb.ProgressBar
}

func (r pgReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		r.pgbar.Add(n)
	}
	return n, err
}

func (d *DockerRegistry) layerVerifyWorker(jobs chan layerDownload, done chan layerDownload) {

	for v := range jobs {
		var dig digest.Digest
		var wrappedPbReader pgReader

		fd, err := os.OpenFile(v.targetPath, os.O_RDONLY, 0644)
		if err != nil {
			v.err = err
			goto theend
		}

		defer fd.Close()

		wrappedPbReader = pgReader{
			Reader: fd,
			pgbar:  v.bar,
		}

		dig, err = v.digest.Algorithm().FromReader(wrappedPbReader)

		if err != nil {
			v.err = err
			goto theend
		}

		if dig.Hex() != v.digest.Hex() {
			v.err = errors.New("digest does not match (" + dig.Hex() + ") != (" + v.digest.Hex() + ")")
			goto theend
		}

	theend:
		done <- v
	}

}

func (d *DockerRegistry) downloadAndVerify(repo string, manifestList []distribution.Descriptor, tmpDir string) error {

	errs := make([]error, 0)
	pool, err := pb.StartPool()

	if err != nil {
		return err
	}

	jobs := make(chan layerDownload, 100)
	downloadDone := make(chan layerDownload, 100)

	verificationJobs := make(chan layerDownload, 100)
	verificationDone := make(chan layerDownload, 100)

	for i := 0; i < 5; i++ {
		go d.layerDownloadWorker(jobs, downloadDone)
		go d.layerVerifyWorker(verificationJobs, verificationDone)
	}

	var c int
	for _, v := range manifestList {
		if v.MediaType != "application/vnd.docker.image.rootfs.diff.tar.gzip" {
			continue
		}
		job := layerDownload{
			size:       v.Size,
			digest:     digest.NewDigestFromHex(v.Digest.Algorithm().String(), v.Digest.Hex()),
			repo:       repo,
			targetPath: filepath.Join(tmpDir, v.Digest.Hex()),
		}

		job.bar = pb.New(1)
		job.bar.Units = pb.U_BYTES
		job.bar.UnitsWidth = 25
		job.bar.ShowSpeed = true
		job.bar.Prefix(v.Digest.Hex()[:Min(len(v.Digest.Hex())-1, 12)] + " ")
		job.bar.ShowCounters = false

		pool.Add(job.bar)
		job.bar.Start()

		jobs <- job
		c++
	}

	close(jobs)

	var dc int
	for dc < c {
		select {
		case msg := <-downloadDone:
			if msg.err != io.EOF {
				dc++
				msg.bar.ShowFinalTime = false
				msg.bar.ShowPercent = false
				msg.bar.ShowCounters = false
				msg.bar.ShowTimeLeft = false
				msg.bar.ShowSpeed = false
				msg.bar.ShowBar = false
				msg.bar.UnitsWidth = 25
				msg.bar.Finish()
				msg.bar.Postfix("[DOWNLOAD ERROR]")
				errs = append(errs, msg.err)
			} else {
				msg.err = nil
				msg.bar.Set(0)
				verificationJobs <- msg
			}
		case msg := <-verificationDone:
			dc++
			msg.bar.ShowFinalTime = false
			msg.bar.ShowPercent = false
			msg.bar.ShowCounters = false
			msg.bar.ShowTimeLeft = false
			msg.bar.ShowSpeed = false
			msg.bar.ShowBar = false
			msg.bar.UnitsWidth = 25
			msg.bar.Finish()
			if msg.err != nil {
				msg.bar.Postfix("[VERIFICATION ERROR] " + msg.err.Error())
				errs = append(errs, msg.err)
			} else {
				msg.bar.Postfix("[OK]")
			}
		}
	}

	close(verificationJobs)
	close(verificationDone)
	close(downloadDone)
	pool.Stop()

	if len(errs) > 0 {
		return errors.New("Error to Download and Verify")
	}

	return nil
}

func (d *DockerRegistry) squashDownloads(manifestList []distribution.Descriptor, tmpDir string, targetFile string) error {

	sqroot := filepath.Join(tmpDir, "squash-root")

	err := os.Mkdir(sqroot, 0755)

	if err != nil {
		return err
	}

	for _, v := range manifestList {
		hexString := v.Digest.Hex()
		tarPath := filepath.Join(tmpDir, hexString)
		fd, err := os.OpenFile(tarPath, os.O_RDONLY, 0644)

		if err != nil {
			return err
		}

		err = Untar(sqroot, fd)
		if err != nil {
			return err
		}
	}

	/*	filepath.Walk(sqroot, func(path string, info os.FileInfo, err error) error {
		baseN := filepath.Base(path)
		dirN := filepath.Dir(path)
		if strings.HasPrefix(baseN, ".wh.") {
			deleteN := baseN[4:]
			deleteP := filepath.Join(dirN, deleteN)
			err := os.RemoveAll(deleteP)
			if err != nil {
				return err
			}
			err = os.RemoveAll(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})*/

	return nil
}

func cleanDir(d string) {

}

func (d *DockerRegistry) ToSquash(repo string, ref string, name string) error {
	if !d.initialized {
		return errors.New("registry not initialized")
	}

	tmpDir, err := ioutil.TempDir("", name+"-")

	log.Println("tmpdir: " + tmpDir)

	if err != nil {
		return err
	}

	defer cleanDir(tmpDir)

	var manifestV2 schema2.Manifest
	var manifestList manifestlist.ManifestList

getmanifest:
	manifest, err := d.registry.Manifest(repo, ref)

	mediaType, _, err := manifest.Payload()

	if mediaType == "application/vnd.docker.distribution.manifest.v2+json" {
		manifestV2, err = d.registry.ManifestV2(repo, ref)
	} else if mediaType == "application/vnd.docker.distribution.manifest.list.v2+json" {
		manifestList, err = d.registry.ManifestList(repo, ref)
		for _, v := range manifestList.Manifests {
			ref = v.Digest.String()
			if v.Platform.Architecture == d.arch {
				goto getmanifest
			}
		}
		if len(manifestList.Manifests) > 0 {
			log.Println("Couldnt find matching arch; using random one")
			goto getmanifest
		} else {
			log.Fatal("Not a single manifest found in manifest list")
		}
	} else {
		log.Fatal("only support schema v2 manifests")
	}

	if err != nil {
		return err
	}

	err = d.downloadAndVerify(repo, manifestV2.Layers, tmpDir)

	if err != nil {
		return err
	}

	err = d.squashDownloads(manifestV2.Layers, tmpDir, d.pvr.Dir)

	if err != nil {
		return err
	}

	return nil
}
