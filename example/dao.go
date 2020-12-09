package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

//const (
//	USERNAME = "quic"
//	PASSWORD = "12345678"
//	NETWORK = "tcp"
//	SERVER = "192.168.1.150"
//	PORT = 3306
//	DATABASE = "quicUpload"
//	)

type FileState struct {
	md5 string
	fileState int
	uuid string
}

func ConnectDatabase() (*sql.DB, error){
	conn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s", USERNAME, PASSWORD, NETWORK, SERVER, PORT, DATABASE)
	fmt.Print(conn)
	connDB, err := sql.Open("mysql", conn)
	if err != nil {
		fmt.Println("Connect mysql error")
		return nil, fmt.Errorf("can not connect database")
	}
	connDB.SetConnMaxIdleTime(100*time.Second)
	connDB.SetMaxOpenConns(100)
	return connDB,nil
}


func InsertDataIntoFileState(DB *sql.DB, state *FileState) {
	_, err := DB.Exec("insert into fileState(file_md5,file_state,uuid) values(?,?,?)",state.md5, state.fileState, state.uuid)
	if err != nil {
		fmt.Printf("Insert data into fileState failed, err：%v\n", err)
		return
	}
	fmt.Print("Insert data into fileState success\n")
}

func QueryDataInFileState(DB *sql.DB, md5 string) (*FileState, error) {
	fileState := new(FileState)
	row := DB.QueryRow("select file_md5,file_state,uuid from fileState where file_md5=?", md5)
	if row.Err() == sql.ErrNoRows {
		return nil,nil
	}
	if err := row.Scan(&fileState.md5, &fileState.fileState,&fileState.uuid); err != nil {
		fmt.Printf("query data in fileState error:%v\n", err)
		return nil, fmt.Errorf("can not find data in database")
	}
	fmt.Println(fileState.md5)
	fmt.Println(fileState.fileState)
	fmt.Println(fileState.uuid)
	return fileState, nil
}

func UpdateDataInFileState(DB *sql.DB,uuid string) {
	_, err := DB.Exec("update fileState set file_state=2 where uuid=?",uuid)
	if err != nil {
		fmt.Printf("update data in fileState error:%v\n", err)
		return
	}
	fmt.Print("Update data in fileState success\n")
}
type FileInfo struct {
	fileMd5 string
	uuid string
	date string
	status string
	name string
	fileLength int
	fileType string
	suffix string
}
func InsertDataInFileInfo(DB *sql.DB, fileInfo *FileInfo) {
	_, err := DB.Exec("insert into fileInfo(file_md5,uuid,date,status,name,file_length,file_type,suffix) values(?,?,?,?,?,?,?,?)",fileInfo.fileMd5,
		fileInfo.uuid,fileInfo.date,fileInfo.status,fileInfo.name,fileInfo.fileLength,fileInfo.fileType,fileInfo.suffix)
	if err != nil {
		fmt.Printf("Insert data into file failed, err：%v\n", err)
		return
	}
	fmt.Print("Insert data into file failed\n")
}


