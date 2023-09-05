/*
 * @Copyright: 
 * @file name: File name
 * @Data: Do not edit
 * @LastEditor: 
 * @LastData: 
 * @Describe: 
 */
package su_mysql

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	slog "go/su_log"
	"go.uber.org/zap"
	//"time"
)

type MysqlClient struct {
	Db      *sqlx.DB
	Uname   string
	Passwd  string
	Addr    string
	DbName  string
	MaxOpenCns int
	MaxIdleCns int
}

func NewMysqlClient(a_uname, a_passwd, a_addr, a_dbname string, a_max_open_conns, a_max_idle_conns int) *MysqlClient{
	return &MysqlClient{Uname: a_uname, Passwd: a_passwd, Addr: a_addr, DbName: a_dbname, MaxOpenCns: a_max_open_conns, MaxIdleCns: a_max_idle_conns}
}

func (mc *MysqlClient)Connect() error{
	var err error
	mc.Db, err = sqlx.Open("mysql", mc.Uname+":"+mc.Passwd+"@tcp("+mc.Addr+")/"+mc.DbName)
	if err != nil {
		slog.Error("mysql 连接failed", zap.Error(err))
		return err
	}
	mc.Db.SetMaxOpenConns(mc.MaxOpenCns)
	mc.Db.SetMaxIdleConns(mc.MaxIdleCns)
	if err := mc.Db.Ping(); err != nil {
		slog.Error("mysql Ping failed", zap.Error(err))
		return err
	}
	return nil
}

func (mc *MysqlClient)Close() {
	err := mc.Db.Close()
	slog.Info("mysql Close ", zap.Error(err))
}

func (mc *MysqlClient)Insert(a_cmd string, a_parm ...interface{}) error {
	r, err := mc.Db.Exec(a_cmd, a_parm)
	if err != nil {
		slog.Error("mysql insert failed", zap.Error(err))
		return err
	}
	id, err := r.LastInsertId()
	if err != nil {
		slog.Error("mysql insert result failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.Any("id", id))
	return nil
}

func (mc *MysqlClient)Update(a_cmd string, a_parm ...interface{}) error {
	r, err := mc.Db.Exec(a_cmd, a_parm)
	if err != nil {
		slog.Error("mysql update failed", zap.Error(err))
		return err
	}
	row, err := r.RowsAffected()
	if err != nil {
		slog.Error("mysql update result failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.Any("row", row))
	return nil
}

func (mc *MysqlClient)Delete(a_cmd string, a_parm ...interface{}) error {
	r, err := mc.Db.Exec(a_cmd, a_parm)
	if err != nil {
		slog.Error("mysql delete failed", zap.Error(err))
		return err
	}
	row, err := r.RowsAffected()
	if err != nil {
		slog.Error("mysql delete result failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.Any("row", row))
	return nil
}

func (mc *MysqlClient)Select(a_dest interface{}, a_cmd string, a_parm ...interface{}) error {
	err := mc.Db.Select(a_dest, a_cmd, a_parm)
	if err != nil {
		slog.Error("mysql query failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.String("a_cmd", a_cmd))
	return nil
}