package common

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/hiddify/hiddify-core/v2/service_manager"
	"github.com/sagernet/sing-box/option"

	"github.com/sagernet/bbolt"
	bboltErrors "github.com/sagernet/bbolt/errors"

	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service/filemanager"
)

var (
	Storage         CacheFile
	bucketExtension = []byte("extension")
	bucketHiddify   = []byte("hiddify")

	bucketNameList = []string{
		string(bucketExtension),
		string(bucketHiddify),
	}
)

type StorageService struct {
	// Storage *CacheFile
}

func (s *StorageService) Start() error {
	Storage = *NewStorage(context.Background(), option.CacheFileOptions{})
	return nil
}

func (s *StorageService) Close() error {
	Storage.DB.Close()
	return nil
}

func init() {
	service_manager.RegisterPreservice(&StorageService{})
}

type CacheFile struct {
	ctx     context.Context
	path    string
	cacheID []byte

	DB *bbolt.DB
}

func NewStorage(ctx context.Context, options option.CacheFileOptions) *CacheFile {
	var path string
	if options.Path != "" {
		path = options.Path
	} else {
		path = "hiddify.db"
	}
	var cacheIDBytes []byte
	if options.CacheID != "" {
		cacheIDBytes = append([]byte{0}, []byte(options.CacheID)...)
	}
	cache := &CacheFile{
		ctx:     ctx,
		path:    filemanager.BasePath(ctx, path),
		cacheID: cacheIDBytes,
	}
	err := cache.start()
	if err != nil {
		log.Panic(err)
	}
	return cache
}

func (c *CacheFile) start() error {
	const fileMode = 0o666
	options := bbolt.Options{Timeout: time.Second}
	var (
		db  *bbolt.DB
		err error
	)
	for i := 0; i < 10; i++ {
		db, err = bbolt.Open(c.path, fileMode, &options)
		if err == nil {
			break
		}
		if errors.Is(err, bboltErrors.ErrTimeout) {
			continue
		}
		if E.IsMulti(err, bboltErrors.ErrInvalid, bboltErrors.ErrChecksum, bboltErrors.ErrVersionMismatch) {
			rmErr := os.Remove(c.path)
			if rmErr != nil {
				return err
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	err = filemanager.Chown(c.ctx, c.path)
	if err != nil {
		db.Close()
		return E.Cause(err, "platform chown")
	}
	err = db.Batch(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			if name[0] == 0 {
				return b.ForEachBucket(func(k []byte) error {
					bucketName := string(k)
					if !(common.Contains(bucketNameList, bucketName)) {
						_ = b.DeleteBucket(name)
					}
					return nil
				})
			} else {
				bucketName := string(name)
				if !(common.Contains(bucketNameList, bucketName)) {
					_ = tx.DeleteBucket(name)
				}
			}
			return nil
		})
	})
	if err != nil {
		db.Close()
		return err
	}
	c.DB = db
	return nil
}

func (c *CacheFile) bucket(t *bbolt.Tx, key []byte) *bbolt.Bucket {
	if c.cacheID == nil {
		return t.Bucket(key)
	}
	bucket := t.Bucket(c.cacheID)
	if bucket == nil {
		return nil
	}
	return bucket.Bucket(key)
}

func (c *CacheFile) createBucket(t *bbolt.Tx, key []byte) (*bbolt.Bucket, error) {
	if c.cacheID == nil {
		return t.CreateBucketIfNotExists(key)
	}
	bucket, err := t.CreateBucketIfNotExists(c.cacheID)
	if bucket == nil {
		return nil, err
	}
	return bucket.CreateBucketIfNotExists(key)
}

func (c *CacheFile) GetExtensionData(extension_id string, default_value any) error {
	err := c.DB.View(func(t *bbolt.Tx) error {
		bucket := c.bucket(t, bucketExtension)
		if bucket == nil {
			return os.ErrNotExist
		}
		setBinary := bucket.Get([]byte(extension_id))
		if len(setBinary) == 0 {
			return os.ErrInvalid
		}
		return json.Unmarshal(setBinary, &default_value)
	})
	return err
}

func (c *CacheFile) SaveExtensionData(extension_id string, data any) error {
	return c.DB.Batch(func(t *bbolt.Tx) error {
		bucket, err := c.createBucket(t, bucketExtension)
		if err != nil {
			return err
		}

		// Assuming T implements MarshalBinary

		setBinary, err := json.MarshalIndent(data, " ", "")
		if err != nil {
			return err
		}
		return bucket.Put([]byte(extension_id), setBinary)
	})
}
