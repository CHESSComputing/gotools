package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Record map[string]any

func decodeDataset(dataset, sep, div string) string {
	arr := strings.Split(dataset, sep)
	rec := make(Record)
	for _, item := range arr {
		if item == "" {
			continue
		}
		kv := strings.Split(item, div)
		val := kv[1]
		if v, err := strconv.Atoi(val); err == nil {
			rec[kv[0]] = v
		} else {
			rec[kv[0]] = kv[1]
		}
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

func encodeDataset(fname, attrs, sep, div string) string {
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	var rec Record
	err = json.Unmarshal(data, &rec)
	if err != nil {
		log.Fatal(err)
	}
	var dset string
	keys := strings.Split(attrs, ",")
	//     var keys []string
	//     for k, _ := range rec {
	//         keys = append(keys, k)
	//     }
	sort.Strings(keys)
	for _, k := range keys {
		v, _ := rec[k]
		dset = fmt.Sprintf("%s%s%s%s%v", dset, sep, k, div, v)
	}
	return dset
}

func main() {
	var encode string
	flag.StringVar(&encode, "encode", "", "json encode")
	var decode string
	flag.StringVar(&decode, "decode", "", "decode string")
	var sep string
	flag.StringVar(&sep, "sep", "/", "attribute separator")
	var div string
	flag.StringVar(&div, "div", ":", "key-value divider")
	var attrs string
	flag.StringVar(&attrs, "attrs", "", "comma separated keys to use for did composition")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var dset string
	if decode != "" {
		dset = decodeDataset(decode, sep, div)
	} else {
		dset = encodeDataset(encode, attrs, sep, div)
	}
	fmt.Println(dset)
}
