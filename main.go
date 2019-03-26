package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)
var url string = "http://www.stats.gov.cn/tjsj/tjbz/tjyqhdmhcxhfdm/2018/index.html"
var baseUrl string = "http://www.stats.gov.cn/tjsj/tjbz/tjyqhdmhcxhfdm/2018/"
var f *os.File
func ConvertToString(src string, srcCode string, tagCode string) string {
	srcCoder := mahonia.NewDecoder(srcCode)
	srcResult := srcCoder.ConvertString(src)
	tagCoder := mahonia.NewDecoder(tagCode)
	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
	result := string(cdata)
	return result
}

func init(){
	f,_=os.Create("area.sql")
}
func MakeSql(sql string)  {
	io.WriteString(f,sql)
}

func GetProvince(provinceChan chan<- map[string]string){
	res,err := http.Get(url)
	res.Close = true
	if err != nil{
		log.Println("请求失败："+err.Error())
		return
	}

	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	doc.Find(".provincetr").Each(func(i int, s *goquery.Selection) {

		s.Find("a").Each(func(i int, s *goquery.Selection) {
			if s.Text() == ""{
				log.Println("省份获取失败")
			}
			var provinceInfo map[string]string
			provinceInfo = make(map[string]string)
			provinceUrl := s.AttrOr("href", "nothing")
			if provinceUrl == "nothing" {
				log.Println("省份url获取失败")
			}
			provinceName := ConvertToString(s.Text(), "gbk", "utf-8")
			nowProvince := string(strings.Split(provinceUrl, ".")[0])
			provinceInfo["provinceUrl"] = baseUrl+provinceUrl
			provinceInfo["provinceName"] = provinceName
			provinceInfo["provinceCode"] = nowProvince
			fmt.Println(provinceInfo["provinceName"])
			provinceChan <- provinceInfo
			sql := "INSERT INTO goals_area(`name`,`code`,`level`,`parent`,`fullpath`) values('"+provinceName+"','"+nowProvince+"',1,0,"+provinceName+");\r\n"
			MakeSql(sql)
			})
	})

}

func GetCity(provinceInfo map[string]string,wg *sync.WaitGroup)  {
	defer wg.Done()
	res,err := http.Get(provinceInfo["provinceUrl"])
	res.Close = true
	if err != nil{
		log.Println("请求失败："+err.Error())
		return
	}

	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	doc.Find(".citytr").Each(func(i int, s *goquery.Selection) {
		nowCity := s.Find("td").First().Text()
		cityName := ConvertToString(s.Find("td").Last().Text(), "gbk", "utf-8")
		cityUrl := baseUrl+s.Find("a").First().AttrOr("href","nothing")
		sql := "INSERT INTO goals_area(`name`,`code`,`level`,`parent`,`fullpath`) values('"+cityName+"','"+nowCity+"',2,"+provinceInfo["provinceCode"]+","+provinceInfo["provinceName"]+cityName+");\r\n"
		MakeSql(sql)
		GetArea(cityUrl,nowCity,cityName,provinceInfo["provinceName"])
		})

}

func GetArea(cityUrl string,cityCode string,cityName string,provinceName string){
	res,err := http.Get(cityUrl)
	res.Close = true
	if err != nil{
		log.Println("请求失败："+err.Error())
		return
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	doc.Find(".countytr").Each(func(i int, s *goquery.Selection) {
		nowCounty := s.Find("td").First().Text()
		if nowCounty != "" {
			countyName := ConvertToString(s.Find("td").Last().Text(), "gbk", "utf-8")
			sql := "INSERT INTO goals_area(`name`,`code`,`level`,`parent`,`fullpath`) values('"+countyName+"','"+nowCounty+"',3,"+cityCode+","+provinceName+cityName+countyName+");\r\n"
			MakeSql(sql)
			log.Println("解析:"+countyName)
		}
	})
}


func main() {
	log.Println("执行开始")
	wg := sync.WaitGroup{}
	cityChan := make(chan map[string]string,100)
	GetProvince(cityChan)
	close(cityChan)
	for provinceInfo := range cityChan{
		wg.Add(1)
		go GetCity(provinceInfo,&wg)
	}

	wg.Wait()
	log.Println("执行结束")
}
