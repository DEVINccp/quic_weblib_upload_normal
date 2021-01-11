package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
)

type Resource struct {
	id int64
	groupId int64
	parentId int64
	resourceType int
	path string
	contentType string
	now time.Time
	groupName string
	memberId int64
	memberName string
	fileOriginalName string
	flag string
	detailSize int64
	size int64
	lastModified string
	resourceStatus string
	rate float64
	suffix string
	filePreName string
	filePath string
	name string
	reserveField int64
}

func GetResourcePathById(DB *sql.DB, parentId string) (*Resource,error) {
	resource := new(Resource)
	row := DB.QueryRow("select id,path from weblib_group_resource where id=?",parentId)
	if row.Err() == sql.ErrNoRows {
		return nil,nil
	}
	if err := row.Scan(&resource.id, &resource.path); err != nil {
		fmt.Printf("query data in weblib_group error:%v\n", err)
		return nil, fmt.Errorf("can not find data in database")
	}
	return resource, nil
}

func saveResourceInfoIntoDatabase(DB *sql.DB, resource *Resource) int64{
	createDate := resource.now.Format("2006-01-02 15:04:05")
	exec, err := DB.Exec("insert into weblib_group_resource(parent_id,group_id,group_name,member_id,member_name,content_type,create_date,original_name,finish_sign,detail_size,"+
		"size,path,type,resource_status,upload_rate,name) values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", resource.parentId, resource.groupId, resource.groupName, resource.memberId, resource.memberName,
		resource.contentType, createDate, resource.fileOriginalName, resource.flag, resource.detailSize, resource.size, resource.path, resource.resourceType, resource.resourceStatus, resource.rate,resource.name)
	resourceId, err := exec.LastInsertId()
	if err != nil {
		fmt.Printf("Insert data into fileState failed, err：%v\n", err)
		return 0
	}
	fmt.Print("update data in weblib_group_resource success\n")
	return resourceId
}

func updateResourceByResourceId(DB *sql.DB, resource *Resource){
	_, err := DB.Exec("update weblib_group_resource set upload_rate=?,finish_sign=?,file_pre_name=?,name=?,file_path=? where id=?",resource.rate,resource.flag,resource.filePreName,resource.name,resource.filePath,resource.id)
	if err != nil {
		fmt.Printf("update resource failed：%v\n", err)
	}
}

func updateResourceReserveFieldByResourceId(DB *sql.DB, resource *Resource) {
	_, err := DB.Exec("update weblib_group_resource set reserve_field1=? where id=?", strconv.FormatInt(resource.reserveField, 10), resource.id)
	if err != nil {
		fmt.Printf("update resource failed：%v\n", err)
	}
}

func queryResourceById(DB *sql.DB, resource *Resource) Resource {
	parentFolder := new(Resource)
	parentFolder.id = resource.parentId
	row := DB.QueryRow("select parent_id,size from weblib_group_resource where id=?", resource.parentId)
	err := row.Scan(&parentFolder.parentId, &parentFolder.size)
	if err != nil {
		panic("No exist resource!")
	}
	return *parentFolder
}

func updateResourceSize(DB *sql.DB, resource *Resource) {
	_, err := DB.Exec("update weblib_group_resource set size=? where id=?", resource.size, resource.id)
if err != nil {
	panic("update resource size failed!")
}
}

func queryResourceByResourceId(DB *sql.DB, resourceId int64) *Resource {
	resource := new(Resource)
	var reserveField string
	row := DB.QueryRow("select group_id,group_name,member_id,member_name,content_type,create_date,original_name,finish_sign,detail_size,"+
		"size,path,type,resource_status,reserve_field1 from weblib_group_resource where id=?", resourceId)
	err := row.Scan(&resource.groupId, &resource.groupName, &resource.memberId, &resource.memberName, &resource.contentType, &resource.now,
		&resource.fileOriginalName, &resource.flag, &resource.detailSize, &resource.size, &resource.path, &resource.resourceType, &resource.resourceStatus, &reserveField)
	parseUint, _ := strconv.ParseInt(reserveField, 10, 64)
	resource.reserveField = parseUint
	if err != nil {
		log.Fatal("query resource failed!\n")
		return nil
	}
	return resource
}

func DeleteResourceInfoByResourceId(DB *sql.DB, resourceId int64){
	_, err := DB.Exec("delete from weblib_group_resource where id=?", resourceId)
	if err != nil {
		panic("delete resource failed!")
	}
}

func updateResourceAtLastChunk(DB *sql.DB, resource *Resource) {
	_, err := DB.Exec("update weblib_group_resource set last_modified=?,reserve_field1=? where id=?", resource.lastModified, strconv.FormatInt(resource.reserveField, 10), resource.id)
	if err != nil {
		panic("update resource failed!")
	}
}

func updateResourceAtMiddle(DB *sql.DB, resource *Resource) {
	_, err := DB.Exec("update weblib_group_resource set last_modified=?,reserve_field1=? where id=?", resource.lastModified, strconv.FormatInt(resource.reserveField, 10), resource.id)
	if err != nil {
		panic("update resource failed!")
	}
}