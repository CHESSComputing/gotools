package main

import (
	"encoding/json"
	"time"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/CHESSComputing/golib/ldap"
)

// UserInfo represents the structure returned by the user service.
type UserInfo struct {
	DN        string    `json:"DN"`
	Name      string    `json:"Name"`
	Email     string    `json:"Email"`
	Uid       string    `json:"Uid"`
	UidNumber int       `json:"UidNumber"`
	GidNumber int       `json:"GidNumber"`
	Groups    []string  `json:"Groups"`
	Btrs      []string  `json:"Btrs"`
	Beamlines []string  `json:"Beamlines"`
	Expire    time.Time `json:"Expire"`
	Foxdens   []string  `json:"Foxdens"`
}

type UserInput struct {
	Name      string
	Email     string
	Uid       string
	UidNumber string
	GidNumber string
}

func userLookup(userInput UserInput) ([]UserInfo, error) {
	var records []ldap.Entry
	// make ldap query
	var entry ldap.Entry
	var err error
	uid := userInput.Uid
	name := userInput.Name
	email := userInput.Email
	uidNumber := userInput.UidNumber
	gid := userInput.GidNumber
	if userInput.Uid != "" {
		entry, err = ldapCache.SearchBy(
			srvConfig.Config.LDAP.Login,
			srvConfig.Config.LDAP.Password,
			uid, "uid")
	} else if gid != "" {
		entry, err = ldapCache.SearchBy(
			srvConfig.Config.LDAP.Login,
			srvConfig.Config.LDAP.Password,
			gid, "gidNumber")
	} else if name != "" {
		entry, err = ldapCache.SearchBy(
			srvConfig.Config.LDAP.Login,
			srvConfig.Config.LDAP.Password,
			name, "name")
	} else if uidNumber != "" {
		entry, err = ldapCache.SearchBy(
			srvConfig.Config.LDAP.Login,
			srvConfig.Config.LDAP.Password,
			uidNumber, "uidNumber")
	} else if email != "" {
		entry, err = ldapCache.SearchBy(
			srvConfig.Config.LDAP.Login,
			srvConfig.Config.LDAP.Password,
			email, "mail")
	}
	var users []UserInfo
	if err != nil {
		return users, err
	}
	records = append(records, entry)
	data, err := json.Marshal(records)
	if err != nil {
		return users, err
	}
	err = json.Unmarshal(data, &users)
	return users, nil
}
