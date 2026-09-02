package app

import "errors"

// ErrRideNotFound 表示查無指定的搭乘紀錄。
var ErrRideNotFound = errors.New("ride record not found")

// ErrConflictAlreadyResolved 表示該筆混車衝突已被裁決過。
var ErrConflictAlreadyResolved = errors.New("conflict already resolved")
