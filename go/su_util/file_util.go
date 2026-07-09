package my_util

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/tealeg/xlsx"
	"io"
	"os"
	"strings"
)

func ReadTxtFile(name string, fn func([]string)) {
	if err := ReadTxtFileE(name, fn); err != nil {
		fmt.Println("ReadTxtFile err:", err)
	}
}

func ReadTxtFileE(name string, fn func([]string)) error {
	fp, err := os.Open(name)
	if err != nil {
		fmt.Println("打开文件错误！！", err)
		return err
	}
	defer fp.Close()
	sc := bufio.NewScanner(fp)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for sc.Scan() {
		strArr := strings.Split(sc.Text(), ",")
		if fn != nil {
			fn(strArr)
		}
	}
	if err = sc.Err(); err != nil {
		return err
	}
	return nil
}

func ReadCSVFile(name string, fn func([]string)) {
	if err := ReadCSVFileE(name, fn); err != nil {
		fmt.Println("ReadCSVFile err:", err)
	}
}

func ReadCSVFileE(name string, fn func([]string)) error {
	fp, err := os.Open(name)
	if err != nil {
		fmt.Println("打开文件错误！！", err)
		return err
	}
	defer fp.Close()
	csvReader := csv.NewReader(fp)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if fn != nil {
			fn(rec)
		}
	}
	return nil
}

func ReadExcelFile(name, sheetName string, fn func([]string)) {
	if err := ReadExcelFileE(name, sheetName, fn); err != nil {
		fmt.Println("ReadExcelFile err:", err)
	}
}

func ReadExcelFileE(name, sheetName string, fn func([]string)) error {
	fp, err := xlsx.OpenFile(name)
	if err != nil {
		fmt.Println("打开文件错误！！", err)
		return err
	}
	for _, sheet := range fp.Sheets {
		if sheet.Name == sheetName {
			for _, row := range sheet.Rows {
				strArr := make([]string, 0)
				for _, cell := range row.Cells {
					strArr = append(strArr, cell.String())
				}
				if fn != nil {
					fn(strArr)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("sheet %q not found", sheetName)
}
