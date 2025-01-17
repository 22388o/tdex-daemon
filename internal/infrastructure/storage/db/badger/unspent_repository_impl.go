package dbbadger

import (
	"context"

	"github.com/dgraph-io/badger/v2"
	"github.com/google/uuid"
	"github.com/tdex-network/tdex-daemon/internal/core/domain"
	"github.com/timshannon/badgerhold/v2"
)

const (
	UnspentBadgerholdKeyPrefix       = "bh_Unspent"
	LockedUnspentBadgerholdKeyPrefix = "bh_LockedUnspent"
)

//badgerhold internal implementation adds prefix to the key
var unspentTablePrefixKey = []byte(UnspentBadgerholdKeyPrefix)

type unspentRepositoryImpl struct {
	store     *badgerhold.Store
	lockStore *badgerhold.Store
}

func NewUnspentRepositoryImpl(store, lockStore *badgerhold.Store) domain.UnspentRepository {
	return unspentRepositoryImpl{store, lockStore}
}

func (u unspentRepositoryImpl) AddUnspents(
	ctx context.Context,
	unspents []domain.Unspent,
) (int, error) {
	return u.addUnspents(ctx, unspents)
}

func (u unspentRepositoryImpl) GetAllUnspents(
	ctx context.Context,
) []domain.Unspent {
	return u.getAllUnspents(ctx)
}

func (u unspentRepositoryImpl) GetAvailableUnspents(
	ctx context.Context,
) ([]domain.Unspent, error) {
	query := badgerhold.Where("Spent").Eq(false).And("Confirmed").Eq(true)
	unlockedOnly := true

	return u.findUnspents(ctx, query, unlockedOnly)
}

func (u unspentRepositoryImpl) GetAllUnspentsForAddresses(
	ctx context.Context, addresses []string,
) ([]domain.Unspent, error) {
	iface := make([]interface{}, 0, len(addresses))
	for _, v := range addresses {
		iface = append(iface, v)
	}

	unlockedOnly := false
	query := badgerhold.Where("Address").In(iface...)

	return u.findUnspents(ctx, query, unlockedOnly)
}

func (u unspentRepositoryImpl) GetAllUnspentsForAddressesAndPage(
	ctx context.Context, addresses []string, page domain.Page,
) ([]domain.Unspent, error) {
	iface := make([]interface{}, 0, len(addresses))
	for _, v := range addresses {
		iface = append(iface, v)
	}

	unlockedOnly := false
	query := badgerhold.Where("Address").In(iface...)
	from := page.Number*page.Size - page.Size
	query.Skip(from).Limit(page.Size)

	return u.findUnspents(ctx, query, unlockedOnly)
}

func (u unspentRepositoryImpl) GetUnspentsForAddresses(
	ctx context.Context,
	addresses []string,
) ([]domain.Unspent, error) {
	iface := make([]interface{}, 0, len(addresses))
	for _, v := range addresses {
		iface = append(iface, v)
	}

	query := badgerhold.Where("Spent").Eq(false).
		And("Confirmed").Eq(true).
		And("Address").In(iface...)
	unlockedOnly := false

	return u.findUnspents(ctx, query, unlockedOnly)
}

func (u unspentRepositoryImpl) GetAvailableUnspentsForAddresses(
	ctx context.Context,
	addresses []string,
) ([]domain.Unspent, error) {
	iface := make([]interface{}, 0, len(addresses))
	for _, v := range addresses {
		iface = append(iface, v)
	}

	query := badgerhold.Where("Spent").Eq(false).
		And("Confirmed").Eq(true).
		And("Address").In(iface...)
	unlockedOnly := true

	return u.findUnspents(ctx, query, unlockedOnly)
}

func (u unspentRepositoryImpl) GetBalance(
	ctx context.Context,
	addresses []string,
	assetHash string,
) (uint64, error) {
	unlockedAndConfirmedOnly := false
	return u.getBalance(ctx, addresses, assetHash, unlockedAndConfirmedOnly)
}

func (u unspentRepositoryImpl) GetUnlockedBalance(
	ctx context.Context,
	addresses []string,
	assetHash string,
) (uint64, error) {
	unlockedAndConfirmedOnly := true
	return u.getBalance(ctx, addresses, assetHash, unlockedAndConfirmedOnly)
}

func (u unspentRepositoryImpl) SpendUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
) (int, error) {
	return u.spendUnspents(ctx, unspentKeys)
}

func (u unspentRepositoryImpl) ConfirmUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
) (int, error) {
	return u.confirmUnspents(ctx, unspentKeys)
}

func (u unspentRepositoryImpl) LockUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
	tradeID uuid.UUID,
) (int, error) {
	return u.lockUnspents(ctx, unspentKeys, tradeID)
}

func (u unspentRepositoryImpl) UnlockUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
) (int, error) {
	return u.unlockUnspents(ctx, unspentKeys)
}

func (u unspentRepositoryImpl) addUnspents(
	ctx context.Context,
	unspents []domain.Unspent,
) (int, error) {
	count := 0
	for _, v := range unspents {
		done, err := u.insertUnspent(ctx, v)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}

	return count, nil
}

func (u unspentRepositoryImpl) getAllUnspents(ctx context.Context) []domain.Unspent {
	unspents, _ := u.findUnspents(ctx, nil, false)
	return unspents
}

func (u unspentRepositoryImpl) getBalance(
	ctx context.Context,
	addresses []string,
	assetHash string,
	unlockedAndConfirmedOnly bool,
) (uint64, error) {
	iface := make([]interface{}, 0, len(addresses))
	for _, v := range addresses {
		iface = append(iface, v)
	}

	query := badgerhold.Where("AssetHash").Eq(assetHash).
		And("Spent").Eq(false).
		And("Address").In(iface...)
	if unlockedAndConfirmedOnly {
		query = query.And("Confirmed").Eq(unlockedAndConfirmedOnly)
	}

	unspents, err := u.findUnspents(ctx, query, unlockedAndConfirmedOnly)
	if err != nil {
		return 0, err
	}

	var balance uint64
	for _, v := range unspents {
		balance += v.Value
	}

	return balance, nil
}

func (u unspentRepositoryImpl) spendUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
) (int, error) {
	count := 0
	for _, key := range unspentKeys {
		done, err := u.spendUnspent(ctx, key)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (u unspentRepositoryImpl) spendUnspent(
	ctx context.Context,
	key domain.UnspentKey,
) (bool, error) {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("VOut").Eq(key.VOut)
	unlockedOnly := false
	unspents, err := u.findUnspents(ctx, query, unlockedOnly)
	if err != nil {
		return false, err
	}

	if unspents == nil {
		return false, nil
	}

	unspent := unspents[0]
	if unspent.Spent {
		return false, nil
	}

	unspent.Spend()
	unspent.Unlock() // prevent conflict, locks not stored under unspent prefix

	if err := u.updateUnspent(ctx, key, unspent); err != nil {
		return false, err
	}

	return true, nil
}

func (u unspentRepositoryImpl) confirmUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
) (int, error) {
	count := 0
	for _, key := range unspentKeys {
		done, err := u.confirmUnspent(ctx, key)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (u unspentRepositoryImpl) confirmUnspent(
	ctx context.Context,
	key domain.UnspentKey,
) (bool, error) {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("VOut").Eq(key.VOut)
	unlockedOnly := true
	unspents, err := u.findUnspents(ctx, query, unlockedOnly)
	if err != nil {
		return false, err
	}

	if unspents == nil {
		return false, nil
	}

	unspent := unspents[0]
	unspent.Confirm()
	unspent.Unlock() // prevent conflict, locks not stored under unspent prefix

	if err := u.updateUnspent(ctx, key, unspent); err != nil {
		return false, err
	}

	return true, nil
}

func (u unspentRepositoryImpl) lockUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
	tradeID uuid.UUID,
) (int, error) {
	count := 0
	for _, key := range unspentKeys {
		done, err := u.lockUnspent(ctx, key, tradeID)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (u unspentRepositoryImpl) lockUnspent(
	ctx context.Context,
	key domain.UnspentKey,
	tradeID uuid.UUID,
) (bool, error) {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("VOut").Eq(key.VOut)
	unlockedOnly := true
	unspents, err := u.findUnspents(ctx, query, unlockedOnly)
	if err != nil {
		return false, err
	}
	if unspents == nil {
		return false, nil
	}

	unspent := unspents[0]
	if unspent.IsLocked() {
		return false, nil
	}

	if err := u.insertLock(ctx, key, tradeID); err != nil {
		return false, err
	}

	return true, nil
}

func (u unspentRepositoryImpl) unlockUnspents(
	ctx context.Context,
	unspentKeys []domain.UnspentKey,
) (int, error) {
	count := 0
	for _, key := range unspentKeys {
		done, err := u.unlockUnspent(ctx, key)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (u unspentRepositoryImpl) unlockUnspent(
	ctx context.Context,
	key domain.UnspentKey,
) (bool, error) {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("VOut").Eq(key.VOut)
	unlockedOnly := false
	unspents, err := u.findUnspents(ctx, query, unlockedOnly)
	if err != nil {
		return false, err
	}

	if unspents == nil {
		return false, nil
	}

	unspent := unspents[0]
	if !unspent.IsLocked() {
		return false, nil
	}

	if err := u.deleteLock(ctx, key); err != nil {
		return false, err
	}

	return true, nil
}

func (u unspentRepositoryImpl) findUnspents(
	ctx context.Context,
	query *badgerhold.Query,
	unlockedOnly bool,
) ([]domain.Unspent, error) {
	var unspents []domain.Unspent
	var unlockedUnspents []domain.Unspent
	var err error
	if ctx.Value("utx") != nil {
		tx := ctx.Value("utx").(*badger.Txn)
		err = u.store.TxFind(tx, &unspents, query)
	} else {
		err = u.store.Find(&unspents, query)
	}
	if err != nil {
		if err == badgerhold.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	for i, unspent := range unspents {
		tradeID, err := u.getLock(ctx, unspent.Key())
		if err != nil {
			return nil, err
		}
		if tradeID != nil {
			unspent.Lock(tradeID)
			unspents[i] = unspent
		}
		if unlockedOnly && !unspent.IsLocked() {
			unlockedUnspents = append(unlockedUnspents, unspent)
		}
	}

	if unlockedOnly {
		return unlockedUnspents, nil
	}
	return unspents, nil
}

func (u unspentRepositoryImpl) updateUnspent(
	ctx context.Context,
	key domain.UnspentKey,
	unspent domain.Unspent,
) error {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("VOut").Eq(key.VOut)
	if ctx.Value("utx") != nil {
		tx := ctx.Value("utx").(*badger.Txn)
		return u.store.TxUpdateMatching(
			tx, &domain.Unspent{}, query, func(record interface{}) error {
				u := record.(*domain.Unspent)
				*u = unspent
				return nil
			},
		)
	}

	return u.store.UpdateMatching(&domain.Unspent{}, query, func(record interface{}) error {
		u := record.(*domain.Unspent)
		*u = unspent
		return nil
	})
}

func (u unspentRepositoryImpl) insertUnspent(
	ctx context.Context,
	unspent domain.Unspent,
) (bool, error) {
	var err error
	if ctx.Value("utx") != nil {
		tx := ctx.Value("utx").(*badger.Txn)
		err = u.store.TxInsert(tx, unspent.Key(), &unspent)
	} else {
		err = u.store.Insert(unspent.Key(), &unspent)
	}

	if err != nil {
		if err == badgerhold.ErrKeyExists {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type LockedUnspent struct {
	TradeID uuid.UUID
}

func (u unspentRepositoryImpl) getLock(
	ctx context.Context,
	key domain.UnspentKey,
) (*uuid.UUID, error) {
	var lockedUnspent LockedUnspent
	var err error

	encKey, err := EncodeKey(key, LockedUnspentBadgerholdKeyPrefix)
	if err != nil {
		return nil, err
	}

	if ctx.Value("tx") != nil {
		tx := ctx.Value("tx").(*badger.Txn)
		err = u.lockStore.TxGet(tx, encKey, &lockedUnspent)
	} else {
		err = u.lockStore.Get(encKey, &lockedUnspent)
	}
	if err != nil {
		if err == badgerhold.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &lockedUnspent.TradeID, nil
}

func (u unspentRepositoryImpl) insertLock(
	ctx context.Context,
	key domain.UnspentKey,
	tradeID uuid.UUID,
) error {
	var err error

	encKey, err := EncodeKey(key, LockedUnspentBadgerholdKeyPrefix)
	if err != nil {
		return err
	}

	if ctx.Value("tx") != nil {
		tx := ctx.Value("tx").(*badger.Txn)
		err = u.lockStore.TxInsert(tx, encKey, LockedUnspent{tradeID})
	} else {
		err = u.lockStore.Insert(encKey, LockedUnspent{tradeID})
	}
	if err != nil {
		if err != badgerhold.ErrKeyExists {
			return err
		}
	}
	return nil
}

func (u unspentRepositoryImpl) deleteLock(
	ctx context.Context,
	key domain.UnspentKey,
) error {
	var err error

	encKey, err := EncodeKey(key, LockedUnspentBadgerholdKeyPrefix)
	if err != nil {
		return err
	}

	if ctx.Value("tx") != nil {
		tx := ctx.Value("tx").(*badger.Txn)
		err = u.lockStore.TxDelete(tx, encKey, LockedUnspent{})
	} else {
		err = u.lockStore.Delete(encKey, LockedUnspent{})
	}
	if err != nil {
		if err != badgerhold.ErrNotFound {
			return err
		}
	}
	return nil
}
