package main

import (
	"encoding/binary"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

const (
	bucket = "voucherBucket"
)

type database struct {
	b *bolt.DB
}

func newDb(path string) (*database, error) {
	db, err := bolt.Open(path, 0600, bolt.DefaultOptions)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	createBucketsFunc := func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return err
	}

	err = db.Update(createBucketsFunc)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("could not init database: %s", err)
	}

	return &database{
		b: db,
	}, nil
}

func (db *database) withdrawMoney(voucher string, cash uint32) (uint32, error) {
	var result uint32
	err := db.b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("expected %s bucket does not exits", bucket)
		}
		var updatedEndingValue uint32

		voucherKey := []byte(voucher)
		val := b.Get(voucherKey)
		if val != nil {
			updatedEndingValue = binary.LittleEndian.Uint32(val)
		}
		updatedEndingValue = updatedEndingValue + cash
		result = updatedEndingValue

		val = make([]byte, 4)
		binary.LittleEndian.PutUint32(val, updatedEndingValue)
		if err := b.Put(voucherKey, val); err != nil {
			return err
		}
		return b.Put(voucherKey, val)
	})
	if err != nil {
		return 0, fmt.Errorf("could not update val in db: %w", err)
	}
	return result, err
}
func (db *database) showMoneyInVoucher(voucher int64) (byte, error) {
	var cash byte

	voucherMoney := func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("expected %s bucket does not exist", bucket)
		}
		voucherKey := make([]byte, 8)
		binary.LittleEndian.PutUint64(voucherKey, uint64(voucher))
		val := b.Get(voucherKey)
		if val != nil {
			cash = val[0]
		}
		return nil
	}
	err := db.b.View(voucherMoney)
	if err != nil {
		return 0, fmt.Errorf("could not check user`s role: %w", err)
	}
	return cash, nil
}

func (db *database) Close() error {
	return db.b.Close()
}
