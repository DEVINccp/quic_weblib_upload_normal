package main

import (
	"database/sql"
	"log"
)

type Group struct {
	id int64
	name string
	displayName string
	path string
	categoryId int64
	groupType string
	availableCapacity int64
	usedCapacity int64
	groupTotalSize int64
}

func getGroupByGroupId(DB *sql.DB, groupId int64) *Group {
	group := new(Group)
	row := DB.QueryRow("select id,name,display_name,path,category_id,type,available_capacity,used_capacity,total_file_size from weblib_group where id=?", groupId)
	err := row.Scan(&group.id, &group.name, &group.displayName, &group.path, &group.categoryId, &group.groupType, &group.availableCapacity, &group.usedCapacity, &group.groupTotalSize)
	if err != nil {
		log.Fatal("query weblib group failed!\n")
		return nil
	}
	return group
}

//In weblib_group table, display_name represents group's name
//name field represents member's ID
func getDisplayNameByGroupId(DB *sql.DB, groupId string) string {
	var displayName string
	row := DB.QueryRow("select display_name from weblib_group where id=?", groupId)
	err := row.Scan(&displayName)
	if err != nil {
		log.Fatal("query display name failed!\n")
		return ""
	}
	return displayName
}

func getGroupByCategoryIdAndGroupDisplayName(DB *sql.DB, categoryId int64, displayName string) *Group{
	group := new(Group)
	row := DB.QueryRow("select id,name,display_name,path,category_id,type,available_capacity,used_capacity,total_file_size from weblib_group where category_id=? and display_name=?", categoryId, displayName)
	err := row.Scan(&group.id, &group.name, &group.displayName, &group.path, &group.categoryId, &group.groupType, &group.availableCapacity, &group.usedCapacity, &group.groupTotalSize)
	if err != nil {
		log.Fatal("query weblib group failed!\n")
		return nil
	}
	return group
}

//In database, memberId correspond to name field in table weblib_group
func getGroupByMemberId(DB *sql.DB, memberId int64) *Group {
	group := new(Group)
	row := DB.QueryRow("select id,name,display_name,path,category_id from weblib_group where name=?", memberId)
	err := row.Scan(&group.id, &group.name, &group.displayName, &group.path, &group.categoryId)
	if err != nil {
		log.Fatal("query weblib group failed!\n")
		return nil
	}
	return group
}

func updateGroupCapacityInfo(DB *sql.DB, group *Group) {
	_, err := DB.Exec("update weblib_group set used_capacity=?,available_capacity=? where id=?", group.usedCapacity, group.availableCapacity, group.id)
	if err != nil {
		log.Fatal("update Group Capacity by group Id!\n")
		return
	}
}