package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

const (
	USERNAME = "root"
	PASSWORD = "ccnl4780#"
	NETWORK = "tcp"
	SERVER = "192.168.1.199"
	PORT = 3306
	DATABASE = "weblibnew"
)

type Member struct {
	id int64
	name string
	account string
}

type Category struct {
	id int64
	name string
	parentId int64
}

type GroupType struct {
	id int64
	singleFileSize int64
}

type DomainCategory struct {
	id int64
	categoryId int64
	domainId int64
	relativePath string
}

type User struct {
	id int64
	account string
}

func ConnectWebLibDatabase() (*sql.DB, error){
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

func addUploadLog(DB *sql.DB, resource *Resource, group *Group, host, userAgent string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	member := getMemberAccountByMemberId(DB, resource.memberId)
	user := getUserIdByAccount(DB, member.account)
	_, err := DB.Exec("insert into weblib_upload_log(account,member_name,member_id,ip,create_date,terminal,target_object,target_object_id,group_name,group_id) values(?,?,?,?,?,?,?,?,?,?)",
		user.account, member.name, member.id, host, now, userAgent, resource.fileOriginalName, resource.id, group.displayName, group.id)
	if err != nil {
		log.Fatal("query resource failed!\n")
		return
	}
}

func getCategoryByCategoryName(DB *sql.DB, categoryName string) *Category{
	fmt.Println(categoryName)
	category := new(Category)
	row := DB.QueryRow("select id,name from weblib_category where name=?", categoryName)
	err := row.Scan(&category.id, &category.name)
	if err != nil {
		log.Fatal("query weblib category failed!\n")
		return nil
	}
	return category
}

//resourcepath: weblib_group_path --> path
func getResourcePathById(DB *sql.DB, id int64) string {
	var path string
	row := DB.QueryRow("select path from weblib_group_resource where id=?", id)
	err := row.Scan(&path)
	if err != nil {
		log.Fatal("query weblib group failed!\n")
		return ""
	}
	return path
}

func getGroupTypeByTypeId(DB *sql.DB, groupTypeId string) *GroupType {
	groupType := new(GroupType)
	row := DB.QueryRow("select id,single_file_size from weblib_group_type where id=?", groupTypeId)
	err := row.Scan(&groupType.id, &groupType.singleFileSize)
	if err != nil {
		log.Fatal("query weblib group failed!\n")
		return nil
	}
	return groupType
}

func checkHaveSameFileInGroupParent(DB *sql.DB, resource *Resource) bool {
	var id int64
	row := DB.QueryRow("select * from weblib_group_resource where group_id=? and name=? and type=? and parent_id=?", resource.groupId, resource.name, resource.resourceType, resource.parentId)
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return false
	}
	return true
}

func getCategoryByGroupCategoryId(DB *sql.DB, categoryId int64) *Category{
	category := new(Category)
	row := DB.QueryRow("select id,parent_id,name from weblib_category where id=?", categoryId)
	err := row.Scan(&category.id, &category.parentId, &category.name)
	if err != nil {
		log.Fatal("query weblib group failed!\n")
		return nil
	}
	return category
}

func getDomainCategoryByCategoryId(DB *sql.DB, categoryId int64) *DomainCategory {
	domainCategory := new(DomainCategory)
	row := DB.QueryRow("select id,relative_path,category_id,domain_id from weblib_domain_category where category_id=?", categoryId)
	err := row.Scan(&domainCategory.id, &domainCategory.relativePath, &domainCategory.categoryId, &domainCategory.domainId)
	if err != nil {
		log.Fatal("query category domain failed!\n")
		return nil
	}
	return domainCategory
}

func getMemberAccountByMemberId(DB *sql.DB, memberId int64) *Member {
	member := new(Member)
	row := DB.QueryRow("select id,name,account from weblib_member where id=?", memberId)
	err := row.Scan(&member.id, &member.name, &member.account)

	if err != nil {
		log.Fatal("query category domain failed!\n")
		return nil
	}
	return member
}

func getUserIdByAccount(DB *sql.DB, account string) *User {
	user := new(User)
	row := DB.QueryRow("select id,account from sys_user where account=?", account)
	err := row.Scan(&user.id, &user.account)
	if err != nil {
		log.Fatal("query category domain failed!\n")
		return nil
	}
	return user
}

func queryMemberIsRoleManager(db *sql.DB, resource *Resource) bool {
	row := db.QueryRow("select * from weblib_member_role where member_id=?", resource.memberId)
	if row.Err() == sql.ErrNoRows {
		return false
	}
	return true
}

func queryMemberIsDomainManager(db *sql.DB, resource *Resource) bool {
	row := db.QueryRow("select * from weblib_domain_manager where manager_id=?", resource.memberId)
	if row.Err() == sql.ErrNoRows {
		return false
	}
	return true
}

func queryMemberIsSystemManager(db *sql.DB, resource *Resource) bool {
	row := db.QueryRow("select * from weblib_admin where member_id=?", resource.memberId)
	if row.Err() == sql.ErrNoRows {
		return false
	}
	return true
}


