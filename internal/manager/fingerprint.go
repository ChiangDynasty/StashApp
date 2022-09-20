package manager

import (
	"errors"
	"fmt"
	"io"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/hash/md5"
	"github.com/stashapp/stash/pkg/hash/oshash"
	"github.com/stashapp/stash/pkg/logger"
)

type fingerprintCalculator struct {
	Config *config.Instance
}

func (c *fingerprintCalculator) calculateOshash(f *file.BaseFile, o file.Opener) (*file.Fingerprint, error) {
	r, err := o.Open()
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	defer r.Close()

	rc, isRC := r.(io.ReadSeeker)
	if !isRC {
		return nil, errors.New("cannot calculate oshash for non-readcloser")
	}

	hash, err := oshash.FromReader(rc, f.Size)
	if err != nil {
		return nil, fmt.Errorf("calculating oshash: %w", err)
	}

	return &file.Fingerprint{
		Type:        file.FingerprintTypeOshash,
		Fingerprint: hash,
	}, nil
}

func (c *fingerprintCalculator) calculateMD5(o file.Opener) (*file.Fingerprint, error) {
	r, err := o.Open()
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	defer r.Close()

	hash, err := md5.FromReader(r)
	if err != nil {
		return nil, fmt.Errorf("calculating md5: %w", err)
	}

	return &file.Fingerprint{
		Type:        file.FingerprintTypeMD5,
		Fingerprint: hash,
	}, nil
}

func (c *fingerprintCalculator) CalculateFingerprints(f *file.BaseFile, o file.Opener, useExisting bool) ([]file.Fingerprint, error) {
	var ret []file.Fingerprint
	calculateMD5 := true

	if isVideo(f.Basename) {
		var (
			fp  *file.Fingerprint
			err error
		)

		if useExisting {
			fp = f.Fingerprints.For(file.FingerprintTypeOshash)
		}

		if fp == nil {
			// calculate oshash first
			fp, err = c.calculateOshash(f, o)
			if err != nil {
				return nil, err
			}
		}

		ret = append(ret, *fp)

		// only calculate MD5 if enabled in config
		// always re-calculate MD5 if the file already has it
		calculateMD5 = c.Config.IsCalculateMD5() || f.Fingerprints.For(file.FingerprintTypeMD5) != nil
	}

	if calculateMD5 {
		var (
			fp  *file.Fingerprint
			err error
		)

		if useExisting {
			fp = f.Fingerprints.For(file.FingerprintTypeMD5)
		}

		if fp == nil {
			if useExisting {
				// log to indicate missing fingerprint is being calculated
				logger.Infof("Calculating checksum for %s ...", f.Path)
			}

			fp, err = c.calculateMD5(o)
			if err != nil {
				return nil, err
			}
		}

		ret = append(ret, *fp)
	}

	return ret, nil
}
