package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

type Updater struct {
	owner   string
	repo    string
	os      string
	arch    string
	version *semver.Version
}

func NewUpdater(owner, repo, os, arch, currVersion string) (*Updater, error) {
	v, err := semver.NewVersion(currVersion)
	if err != nil {
		return nil, errors.New("current version is not valid semver")
	}
	// according to replacements in .goreleaser.yml
	if arch == "amd64" {
		arch = "x86_64"
	}
	if arch == "windows" {
		return nil, errors.New("windows selfupdate temporarily not supported")
	}
	return &Updater{
		owner:   owner,
		repo:    repo,
		os:      os,
		arch:    arch,
		version: v,
	}, nil
}

func (upd *Updater) Update() error {
	fmt.Printf("Looking for a new version for %s_%s\n", upd.os, upd.arch)

	latest, err := getLatestRelease(upd.owner, upd.repo)
	if err != nil {
		return errors.Wrap(err, "get latest release")
	}
	if len(latest.Assets) == 0 {
		return errors.New("no assets found for the latest releas")
	}
	need, err := upd.needUpdate(latest.TagName)
	if err != nil {
		return errors.Wrap(err, "check need for update")
	}
	if !need {
		fmt.Println("Ferret is up to date")
		return nil
	}

	fmt.Println("New version of Ferret available!")
	fmt.Println("Update Ferret to", latest.TagName)
	fmt.Println("Download checksums and assets")

	sha256, err := upd.downloadChecksum(latest.Assets)
	if err != nil {
		return errors.Wrap(err, "download checksum")
	}
	compressed, asset, err := upd.downloadBin(latest.Assets)
	if err != nil {
		return errors.Wrap(err, "download bin")
	}

	fmt.Println("Verify checksum")

	if err = verifyBin(sha256, compressed); err != nil {
		return errors.Wrap(err, "verify checksum")
	}

	fmt.Println("Uncompress", asset.Name)

	bin, err := uncompress(compressed, binType(asset.Name))
	if err != nil {
		return errors.Wrap(err, "uncompress bin")
	}

	fmt.Println("Install new version")

	if err = replaceBin(bin); err != nil {
		return errors.Wrap(err, "replace old and new bin")
	}
	return nil
}

func (upd *Updater) needUpdate(latest string) (bool, error) {
	latestV, err := semver.NewVersion(latest)
	if err != nil {
		return false, errors.Wrap(err, "latest version is not valid semver")
	}
	return latestV.Compare(upd.version) == 1, nil
}

func (upd *Updater) downloadChecksum(assets []releaseAsset) (sha256 [sha256.Size]byte, err error) {
	assetID := int64(-1)
	for _, asset := range assets {
		if asset.Name == "cli_checksums.txt" {
			assetID = asset.ID
		}
	}
	if assetID == -1 {
		return sha256, errors.New("checksum asset not found")
	}

	asset, err := getReleaseAsset(upd.owner, upd.repo, assetID)
	if err != nil {
		return sha256, errors.Wrap(err, "get asset")
	}
	defer asset.Close()

	data, err := io.ReadAll(asset)
	if err != nil {
		return sha256, errors.Wrap(err, "read asset")
	}
	sha256, err = platformChecksum(data, upd.os, upd.arch)
	if err != nil {
		return sha256, errors.Wrap(err, "find platform checksum")
	}
	return sha256, nil
}

func platformChecksum(checksums []byte, os, arch string) ([sha256.Size]byte, error) {
	sum256 := [sha256.Size]byte{}
	osArch := os + "_" + arch

	// file `checksums` looks something like this:
	//   sha256_hex ferret_darwin_x86_64
	//   sha256_hex ferret_linux_arm64
	//   ...
	for _, line := range bytes.Split(checksums, []byte("\n")) {
		if bytes.Contains(line, []byte(osArch)) {
			segments := bytes.Split(line, []byte(" "))
			if len(segments) == 0 {
				return sum256, errors.New("invalid checksums file")
			}
			_, err := hex.Decode(sum256[:], segments[0])
			if err != nil {
				return sum256, errors.Wrap(err, "invalid hex string")
			}
			return sum256, nil
		}
	}

	return sum256, errors.Errorf("no checksum found for %s", osArch)
}

func (upd *Updater) downloadBin(assets []releaseAsset) ([]byte, *releaseAsset, error) {
	osArch := upd.os + "_" + upd.arch
	binAsset := releaseAsset{}
	for _, asset := range assets {
		if strings.Contains(asset.Name, osArch) {
			binAsset = asset
			break
		}
	}
	if binAsset.Name == "" {
		return nil, nil, errors.New("bin asset not found")
	}

	rd, err := getReleaseAsset(upd.owner, upd.repo, binAsset.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get asset")
	}
	defer rd.Close()

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, nil, errors.Wrap(err, "read asset")
	}
	return data, &binAsset, nil
}

func verifyBin(checksum [sha256.Size]byte, bin []byte) error {
	sum := sha256.Sum256(bin)
	if sum == checksum {
		return nil
	}
	return errors.Errorf("Invalid checksum\n  expected: %x\n  actual:   %x", checksum, sum)
}

const (
	tgzType = "tgz"
	zipType = "zip"

	binName = "ferret"
)

func binType(name string) string {
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		return tgzType
	default:
		// cut `.` at the beginning of the string
		return filepath.Ext(name)[1:]
	}
}

func uncompress(data []byte, contentType string) ([]byte, error) {
	var rd io.Reader

	switch contentType {
	case tgzType:
		gziprd, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, errors.Wrap(err, "new gzip reader")
		}
		defer gziprd.Close()

		tgzrd := tar.NewReader(gziprd)
		for {
			hdr, err := tgzrd.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, errors.Wrap(err, "uncompress tgz")
			}
			if hdr.Name == binName {
				rd = tgzrd
				break
			}
		}

	case zipType:
		return nil, errors.New("zip files are temporarily not supported")

	default:
		return nil, errors.Errorf("unknown content type \"%s\"", contentType)
	}

	if rd == nil {
		return nil, errors.New("bin file not found in acrhive")
	}
	return io.ReadAll(rd)
}

func replaceBin(newbin []byte) error {
	currpath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "get executable path")
	}

	currfile := filepath.Base(currpath)
	currdir := filepath.Dir(currpath)
	prevpath := filepath.Join(currdir, currfile+".prev")

	// move current bin into bin.prev
	err = os.Rename(currpath, prevpath)
	if err != nil {
		return errors.Wrap(err, "move current bin")
	}

	// write new bin
	err = os.WriteFile(currpath, newbin, 0777)
	if err != nil {
		// try to rollback
		if rberr := os.Rename(prevpath, currpath); rberr != nil {
			return errors.Wrapf(rberr, "selfupdate and rollback are failed. bin moved to %s", prevpath)
		}

		return errors.Wrap(err, "selfupdate failed. rollback done")
	}

	err = os.Remove(prevpath)
	if err != nil {
		fmt.Printf("selfupdate done, but old program is not deleted: %s\n", prevpath)
	}

	return nil
}
