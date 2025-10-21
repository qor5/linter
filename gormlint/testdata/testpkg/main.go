package testpkg

import (
	"context"
	"net/http"

	"gorm.io/gorm"
)

type User struct{ ID int }

func good(db *gorm.DB) {
	var u User
	db.First(&u)
}

func bad(db *gorm.DB) {
	var u User
	db.First(&u, 10) // want "gorm DB.First must be called with exactly 1 argument"
}

func okWithContextFirst(db *gorm.DB, ctx context.Context) {
	var u User
	db.WithContext(ctx).First(&u)
}

func okWithRequest(db *gorm.DB, req *http.Request) {
	var u User
	db.WithContext(req.Context()).Find(&u)
}

func okPreparedVar(db *gorm.DB, ctx context.Context) {
	db2 := db.WithContext(ctx)
	db2.Exec("SELECT 1")
}

func okSession(db *gorm.DB, ctx context.Context) {
	db.Session(&gorm.Session{Context: ctx}).Create(&User{ID: 1})
}

func okTransaction(db *gorm.DB, ctx context.Context) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u User
		return tx.Save(&u).Error
	})
}

func badNoContext(db *gorm.DB, ctx context.Context) {
	var u User
	db.First(&u) // want "first call on gorm DB must be WithContext"
}

func badLateWithContext(db *gorm.DB, ctx context.Context) {
	var u User
	db.Model(&u).WithContext(ctx).Find(&u) // want "first call on gorm DB must be WithContext"
}

func badBackground(db *gorm.DB, ctx context.Context) {
	var u User
	db.WithContext(context.Background()).First(&u) // want "do not use context.Background/TODO"
}

func badTODO(db *gorm.DB, ctx context.Context) {
	var u User
	db.WithContext(context.TODO()).First(&u) // want "do not use context.Background/TODO"
}

func badTransactionNoContext(db *gorm.DB, ctx context.Context) error {
	return db.Transaction(func(tx *gorm.DB) error { // want "first call on gorm DB must be WithContext"
		var u User
		return tx.Find(&u).Error
	})
}

func okCtxAlias(db *gorm.DB, ctx context.Context) {
	c := ctx
	db.WithContext(c).Exec("SELECT 1")
}

// No provider param: no enforcement
func okNoProvider(db *gorm.DB) {
	var u User
	db.Model(&u).Where("id = ?", 1).Find(&u)
}

// No provider param: Background is allowed
func okNoProviderBackground(db *gorm.DB) {
	var u User
	db.WithContext(context.Background()).First(&u)
}

// Session without Context field literal should be invalid
func badSessionNoContextField(db *gorm.DB, ctx context.Context) {
	var u User
	db.Session(&gorm.Session{}).First(&u) // want "first call on gorm DB must be WithContext"
}

// Session with variable literal (not inspected) should be invalid
func badSessionVarContext(db *gorm.DB, ctx context.Context) {
	var u User
	s := &gorm.Session{Context: ctx}
	db.Session(s).First(&u) // want "first call on gorm DB must be WithContext"
}

// Function returning DB: first method must still be WithContext
func okFromFuncReturningDB(factory func() *gorm.DB, ctx context.Context) {
	var u User
	factory().WithContext(ctx).First(&u)
}

func badFromFuncReturningDB(factory func() *gorm.DB, ctx context.Context) {
	var u User
	factory().Model(&u).Find(&u) // want "first call on gorm DB must be WithContext"
}

// First-arg rule still applies even if WithContext is correct
func badFirstArgsAfterWithContext(db *gorm.DB, ctx context.Context) {
	var u User
	db.WithContext(ctx).First(&u, 1) // want "gorm DB.First must be called with exactly 1 argument"
}

// no runtime setup needed for analysistest
