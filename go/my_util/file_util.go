package my_util

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/tealeg/xlsx"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ReadTxtFile(name string, fn func([]string)) {
	fp, err := os.Open(name)
	if err != nil {
		fmt.Println("打开文件错误！！", err)
		log.Fatal(err)
		return
	}
	defer fp.Close()
	sc := bufio.NewScanner(fp)
	for sc.Scan() {
		fmt.Println("line:", sc.Text())
		strArr := strings.Split(sc.Text(), ",")
		fn(strArr)
	}
	if err = sc.Err(); err != nil {
		log.Fatal(err)
	}
}

func ReadCSVFile(name string, fn func([]string)) {
	fp, err := os.Open(name)
	if err != nil {
		fmt.Println("打开文件错误！！", err)
		log.Fatal(err)
		return
	}
	defer fp.Close()
	csvReader := csv.NewReader(fp)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("line: ", rec)
		fn(rec)
	}
}

func ReadExcelFile(name, sheetName string, fn func([]string)) {
	fp, err := xlsx.OpenFile(name)
	if err != nil {
		fmt.Println("打开文件错误！！", err)
		log.Fatal(err)
		return
	}
	for _, sheet := range fp.Sheets {
		fmt.Println(sheet.Name)
		if sheet.Name == sheetName {
			for _, row := range sheet.Rows {
				strArr := make([]string, 0)
				for _, cell := range row.Cells {
					strArr = append(strArr, cell.String())
				}
				fmt.Println("line ", row.Cells, strArr)
				fn(strArr)
			}
			break
		}
	}
}