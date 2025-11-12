package model

import (
	"database/sql/driver"
	"errors"
	"log"
	"regexp"

	gosqlite "github.com/glebarez/go-sqlite"
)

// 为 sqlite 定义一个用于正则表达式匹配的函数
// SQL 的 "X regexp Y" 语法会调用 regexp(Y, X)
func regexpFunc(fnCtx *gosqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	// pattern, s := args[0].(string), args[1].(string)
	pattern, ok := args[0].(string)
	if !ok {
		return nil, errors.New("pattern must be a string")
	}
	s, ok := args[1].(string)
	if !ok {
		return nil, errors.New("s must be a string")
	}
	match, err := regexp.MatchString(pattern, s)
	if err != nil {
		return nil, err
	}
	return match, nil
}

func init() {
	err := gosqlite.RegisterDeterministicScalarFunction("regexp", 2, regexpFunc)
	if err != nil {
		log.Fatalf("无法注册 regexp 函数: %v", err)
	}
}