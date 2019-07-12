// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/liquidata-inc/ld/dolt/go/store/d"
)

const tempTablePrefix = "nbs_table_"

func newFSTablePersister(dir string, fc *fdCache, indexCache *indexCache) tablePersister {
	d.PanicIfTrue(fc == nil)
	return &fsTablePersister{dir, fc, indexCache}
}

type fsTablePersister struct {
	dir        string
	fc         *fdCache
	indexCache *indexCache
}

func (ftp *fsTablePersister) Open(ctx context.Context, name addr, chunkCount uint32, stats *Stats) (chunkSource, error) {
	return newMmapTableReader(ftp.dir, name, chunkCount, ftp.indexCache, ftp.fc)
}

func (ftp *fsTablePersister) Persist(ctx context.Context, mt *memTable, haver chunkReader, stats *Stats) (chunkSource, error) {
	name, data, chunkCount, err := mt.write(haver, stats)

	if err != nil {
		return emptyChunkSource{}, err
	}

	return ftp.persistTable(ctx, name, data, chunkCount, stats)
}

func (ftp *fsTablePersister) persistTable(ctx context.Context, name addr, data []byte, chunkCount uint32, stats *Stats) (cs chunkSource, err error) {
	if chunkCount == 0 {
		return emptyChunkSource{}, nil
	}

	tempName, err := func() (tempName string, ferr error) {
		var temp *os.File
		temp, ferr = ioutil.TempFile(ftp.dir, tempTablePrefix)

		if ferr != nil {
			return "", ferr
		}

		defer func() {
			closeErr := temp.Close()

			if ferr == nil {
				ferr = closeErr
			}
		}()

		_, ferr = io.Copy(temp, bytes.NewReader(data))

		if ferr != nil {
			return "", ferr
		}

		index, ferr := parseTableIndex(data)

		if ferr != nil {
			return "", ferr
		}

		if ftp.indexCache != nil {
			ftp.indexCache.lockEntry(name)
			defer func() {
				unlockErr := ftp.indexCache.unlockEntry(name)

				if ferr == nil {
					ferr = unlockErr
				}
			}()
			ftp.indexCache.put(name, index)
		}

		return temp.Name(), nil
	}()

	if err != nil {
		return nil, err
	}

	newName := filepath.Join(ftp.dir, name.String())
	err = ftp.fc.ShrinkCache()

	if err != nil {
		return nil, err
	}

	err = os.Rename(tempName, newName)

	if err != nil {
		return nil, err
	}

	return ftp.Open(ctx, name, chunkCount, stats)
}

func (ftp *fsTablePersister) ConjoinAll(ctx context.Context, sources chunkSources, stats *Stats) (chunkSource, error) {
	plan, err := planConjoin(sources, stats)

	if err != nil {
		return emptyChunkSource{}, err
	}

	if plan.chunkCount == 0 {
		return emptyChunkSource{}, nil
	}

	name := nameFromSuffixes(plan.suffixes())
	tempName, err := func() (tempName string, ferr error) {
		var temp *os.File
		temp, ferr = ioutil.TempFile(ftp.dir, tempTablePrefix)

		if ferr != nil {
			return "", ferr
		}

		defer func() {
			closeErr := temp.Close()

			if ferr == nil {
				ferr = closeErr
			}
		}()

		for _, sws := range plan.sources.sws {
			var r io.Reader
			r, ferr = sws.source.reader(ctx)

			if ferr != nil {
				return "", ferr
			}

			n, ferr := io.CopyN(temp, r, int64(sws.dataLen))

			if ferr != nil {
				return "", ferr
			}

			if uint64(n) != sws.dataLen {
				return "", errors.New("failed to copy all data")
			}
		}

		_, ferr = temp.Write(plan.mergedIndex)

		if ferr != nil {
			return "", ferr
		}

		var index tableIndex
		index, ferr = parseTableIndex(plan.mergedIndex)

		if ferr != nil {
			return "", ferr
		}

		if ftp.indexCache != nil {
			ftp.indexCache.put(name, index)
		}

		return temp.Name(), nil
	}()

	if err != nil {
		return nil, err
	}

	err = os.Rename(tempName, filepath.Join(ftp.dir, name.String()))

	if err != nil {
		return nil, err
	}

	return ftp.Open(ctx, name, plan.chunkCount, stats)
}