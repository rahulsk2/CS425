package main

import "sort"

type Command struct {
	cmd int //0 - Insert, 1 - Delete, 2 - ReadAll
	ID string
}

/*
This is used as an interface to interact with the membershiplist
 */
func memberList() {
	var membershipList []string
	for {
		select {
		case command:=<-mListInput:
			if command.cmd==0 {
				membershipList=insertInSortedMembershipList(command.ID,membershipList)
				mListOutput <-[]string{}
			} else if command.cmd==1 { //delete node
				flag:=true
				for i, v := range membershipList {
					if v==command.ID {
						if i!=len(membershipList)-1 {
							membershipList=append(membershipList[:i],membershipList[i+1:]...)
						} else {
							membershipList=membershipList[:i]
						}
						mListOutput <-[]string{"TRUE"}
						flag=false
						break
					}
				}
				if flag {
					mListOutput <-[]string{"FALSE"}
				}
			} else {
				newSlice := make([]string,len(membershipList))
				copy(newSlice,membershipList)
				mListOutput <-newSlice
			}
		}
	}
}

func insertInSortedMembershipList(idToAdd string, existingList []string) []string{
	newList := append(existingList,idToAdd)
	sort.Strings(newList)
	return newList
}

//TODO: Add logic to membership node insert, delete, readAll
func MembershipListReadAll() []string{
	mListInput <- Command {cmd: 2, ID:""}
	mList:= <- mListOutput
	return mList
}

func MembershipListInsert(v string){
	mListInput <- Command{cmd: 0,ID:v}
	<-mListOutput
}

func MembershipListDelete(v string) []string{
	mListInput <-Command{cmd: 1,ID:v}
	result:=<-mListOutput
	return result
}

