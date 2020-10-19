package main

import (
	"bufio"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	var countEuropeana = 0

	// Create the file to put the info
	f, err := os.Create("tiers.csv")
	check(err)

	defer f.Close()

	f.Sync()

	var w = bufio.NewWriter(f)

	// Write the file headers
	w.WriteString("Repository No.,Resource Type,publication Date,Publication Language,Title,Access Rights,Entity,File Type, Object Size, isShownBy - Size,PDF Downloading Time,Tier\n")

	for num := 1; num < 300000; num++ {

		urlEdm := fmt.Sprintf("%s%d%s", "EDM repository URL", num, "&metadataPrefix=edm")
		urlMarcxchange := fmt.Sprintf("%s%d%s", "MarcXchange repository URL", num, "&metadataPrefix=marcxchange")

		// First check the MarcXchange repository, find the Unimarc 958$d field value, and if it is "1", then it is a Europeana record
		if isEuropeana(urlMarcxchange) == true {

			number := fmt.Sprintf("%v", num)
			fmt.Printf("\n %v", number)
			w.WriteString("\n" + number)

			// if a field doesn't have any information then it has tier zero
			count := metaData(urlEdm, w)

			if count < 6 {
				w.WriteString("")
				w.Flush()
			} else {
				isShownByFileType := getIsShownByFileType(urlEdm, w)
				object := getObject(urlEdm, w)
				start := time.Now()
				isShownByFileSize := getIsShownByFileSize(urlEdm, w)
				rights := rights(urlEdm, w)
				edmType := edmType(urlEdm)
				isShownAt := workingIsShownAt(urlEdm)
				isShownBy := workingIsShownBy(urlEdm)

				// count the time the file takes to download so that if it is greater than 20 minutes it has tier zero
				t := time.Now()
				elap := t.Sub(start)
				elap = elap / time.Minute
				elapsed := int(elap)
				elapsedString := fmt.Sprintf("%v", elapsed)

				// Obtain the tier
				if (edmType == "IMAGE" && isShownByFileType == "jpeg" && isShownByFileSize >= 0.95 && rights == "http://creativecommons.org/publicdomain/mark/1.0/") || ((edmType == "TEXT" && isShownByFileType == "pdf" && rights == "http://creativecommons.org/publicdomain/mark/1.0/") || (edmType == "TEXT" && isShownByFileType == "jpeg" && isShownByFileSize >= 0.95 && rights == "http://creativecommons.org/publicdomain/mark/1.0/")) {
					if elapsed > 20 {
						w.WriteString("," + elapsedString + ",0")
					}
					w.WriteString("," + elapsedString + ",4")
				} else if (edmType == "IMAGE" && isShownByFileType == "jpeg" && isShownByFileSize >= 0.95 && rights == "http://rightsstatements.org/vocab/InC/1.0/") || ((edmType == "TEXT" && isShownByFileType == "pdf" && rights == "http://rightsstatements.org/vocab/InC/1.0/") || (edmType == "TEXT" && isShownByFileType == "jpeg" && isShownByFileSize >= 0.95 && rights == "http://rightsstatements.org/vocab/InC/1.0/")) {
					if elapsed > 20 {
						w.WriteString("," + elapsedString + ",0")
					}
					w.WriteString("," + elapsedString + ",3")
				} else if (edmType == "IMAGE" && isShownByFileType == "jpeg" && (isShownByFileSize >= 0.42 && isShownByFileSize < 0.95)) || ((edmType == "TEXT" && isShownByFileType == "pdf") || (edmType == "TEXT" && isShownByFileType == "jpeg" && (isShownByFileSize >= 0.42 && isShownByFileSize < 0.95))) {
					if elapsed > 20 {
						w.WriteString("," + elapsedString + ",0")
					}
					w.WriteString("," + elapsedString + ",2")
				} else if ((edmType == "IMAGE" && isShownBy == true && isShownByFileType == "jpeg" && (isShownByFileSize >= 0.1 && isShownByFileSize < 0.42)) || (edmType == "IMAGE" && isShownBy == true && (object >= 0.1))) || (edmType == "TEXT" && isShownAt == true) {
					if elapsed > 20 {
						w.WriteString("," + elapsedString + ",0")
					}
					w.WriteString("," + elapsedString + ",1")
				} else {
					w.WriteString("," + elapsedString + ",0")
				}
				w.Flush()

				countEuropeana++
			}
		}
	}
}

// Check if the record is to be exported to Europeana
func isEuropeana(urlMarcxchange string) bool {
	var (
		question  bool
		field958d string
		countA    int
	)

	res, err := http.Get(urlMarcxchange)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	// Get the 958$d field value and check if it has the value "1". If it has, then the record is to exported to Europeana
	doc.Find("mx\\:datafield").Each(func(i int, s *goquery.Selection) {
		tag, _ := s.Attr("tag")
		if tag == "958" {
			s.Find("mx\\:subfield").Each(func(i int, e *goquery.Selection) {
				if attr, _ := e.Attr("code"); attr == "d" {
					if countA == 0 {
						field958d = e.Text()
					}
					countA++
				}
				if field958d == "1" {
					question = true
				} else {
					question = false
				}
			})
		}
	})
	return question
}

func workingIsShownAt(urlEdm string) bool {

	var boolIsShownAt bool

	res, err := http.Get(urlEdm)
	if err != nil {
		fmt.Println(err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	isShown := doc.Find("edm\\:isShownAt")
	isShownAt, _ := isShown.Attr("rdf:resource")

	if isShownAt != "" {
		boolIsShownAt = checkIsShownBy(isShownAt)
	} else {
		boolIsShownAt = false
	}
	return boolIsShownAt
}

func workingIsShownBy(urlEdm string) bool {

	var boolIsShownBy bool

	res, err := http.Get(urlEdm)
	if err != nil {
		fmt.Println(err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	isShown := doc.Find("edm\\:isShownBy")
	isShownBy, _ := isShown.Attr("rdf:resource")

	if isShownBy != "" {
		boolIsShownBy = checkIsShownBy(isShownBy)
	} else {
		boolIsShownBy = false
	}
	return boolIsShownBy
}

func checkIsShownBy(url string) bool {
	var boolShown bool

	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
		boolShown = false
	} else {
		boolShown = true
	}
	return boolShown
}

// Get the rights statement
func rights(urlEdm string, w *bufio.Writer) string {

	res, err := http.Get(urlEdm)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	rig := doc.Find("edm\\:rights")
	rights, _ := rig.Attr("rdf:resource")

	return rights
}

// Get the metadata
func metaData(urlEdm string, w *bufio.Writer) int {

	var (
		resourceType, publicationDate, publicationLanguage, title, entity string
		count                                                             int
	)

	res, err := http.Get(urlEdm)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	// Check if edm\\:type exists and write the correspondent value
	doc.Find("edm\\:type").Each(func(i int, s *goquery.Selection) {
		resourceType = fmt.Sprintf("%s", s.Text())
		if resourceType != "" {
			if resourceType == "TEXT" || resourceType == "IMAGE" || resourceType == "VIDEO" || resourceType == "SOUND" {
				w.WriteString("," + resourceType)
				w.Flush()
				count++
			}
		} else {
			w.WriteString(",Does not have the Resource Type")
			w.Flush()
		}
	})
	// Check if edm\\:issued exists and write the correspondent value
	doc.Find("dcterms\\:issued").Each(func(i int, s *goquery.Selection) {
		publicationDate = fmt.Sprintf("%s", s.Text())
		if publicationDate != "" {
			w.WriteString("," + publicationDate)
			w.Flush()
			count++

		} else {
			w.WriteString(",Does not have the Publication Date")
			w.Flush()
		}
	})
	// Check if edm\\:language exists and write the correspondent value
	doc.Find("dc\\:language").Each(func(i int, s *goquery.Selection) {
		publicationLanguage = fmt.Sprintf("%s", s.Text())
		if publicationLanguage != "" {
			w.WriteString("," + publicationLanguage)
			w.Flush()
			count++

		} else {
			w.WriteString(",Does not have the Language")
			w.Flush()
		}
	})
	// Check if edm\\:title exists and write the correspondent value
	doc.Find("dc\\:title").Each(func(i int, s *goquery.Selection) {
		title = fmt.Sprintf("%s", s.Text())
		if title != "" {
			w.WriteString(",Has a title")
			w.Flush()
			count++

		} else {
			w.WriteString(",Does not have the Title")
			w.Flush()
		}
	})
	// Check if edm\\:rights exists and write the correspondent value
	doc.Find("edm\\:rights").Each(func(i int, s *goquery.Selection) {
		rights, ok := s.Attr("rdf:resource")
		if ok != false {
			w.WriteString("," + rights)
			w.Flush()
			count++
		} else {
			w.WriteString(",Does not have the Rights")
			w.Flush()
		}
	})
	// Check if edm\\:dataProvider exists and write the correspondent value
	doc.Find("edm\\:dataProvider").Each(func(i int, s *goquery.Selection) {
		entity = fmt.Sprintf("%s", s.Text())
		if entity != "" {
			w.WriteString("," + entity)
			w.Flush()
			count++

		} else {
			w.WriteString(",Does not have Entity")
			w.Flush()
		}
	})
	return count
}

func getIsShownByFileSize(urlEdm string, w *bufio.Writer) float64 {

	var url string
	var fileSize float64

	response, err := http.Get(urlEdm)

	if err != nil {
		fmt.Printf("%s", err)
	}

	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Printf("%s", err)
	}

	defer response.Body.Close()

	content := string(contents)
	b := strings.Index(content, "<edm:isShownBy rdf:resource=")
	e1 := strings.Index(content, "\"/><edm:hasView rdf:resource")
	e2 := strings.Index(content, "\"/><edm:object rdf:resource")

	// if e < b then e it is not after b
	if b != -1 && e1 != -1 && e1 > b {
		x := content[b:e1]
		url = fmt.Sprintf("%s", x[29:])
		fileSize = getFileSize(url, w)
	} else if b != -1 && e2 != -1 && e2 > b {
		x := content[b:e2]
		url = fmt.Sprintf("%s", x[29:])
		fileSize = getFileSize(url, w)
	} else {
		w.WriteString(",Does not have isShownBy")
		w.Flush()
	}

	return fileSize
}

// Get the isShownBy
func getIsShownByFileType(urlEdm string, w *bufio.Writer) string {

	var url string
	var fileType string

	response, err := http.Get(urlEdm)

	if err != nil {
		fmt.Printf("%s", err)
	}

	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()

		content := string(contents)
		b := strings.Index(content, "<edm:isShownBy rdf:resource=")
		e1 := strings.Index(content, "\"/><edm:hasView rdf:resource")
		e2 := strings.Index(content, "\"/><edm:object rdf:resource")

		// if e < b then e it is not after b
		if b != -1 && e1 != -1 && e1 > b {
			x := content[b:e1]
			url = fmt.Sprintf("%s", x[29:])
			fileType = getFileType(url, w)
		} else if b != -1 && e2 != -1 && e2 > b {
			x := content[b:e2]
			url = fmt.Sprintf("%s", x[29:])
			fileType = getFileType(url, w)
		} else {
			w.WriteString(",Does not have the isShownBy field")
			w.Flush()
		}
	}
	return fileType
}

// Get the file type
func getFileType(url string, w *bufio.Writer) string {

	var filetype string
	resp, err := http.Get(url)

	if err != nil {
		fmt.Printf("%s", err)
	}

	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	buff := make([]byte, 256) // take only first 256 bytes into consideration

	if _, err = reader.Read(buff); err != nil {
		fmt.Println(err)
		erro := fmt.Sprintf("%s", err)
		w.WriteString(",Erro: " + erro)
	} else {
		contentype := http.DetectContentType(buff)
		subStringsSlice := strings.Split(contentype, "/")
		filetype = subStringsSlice[len(subStringsSlice)-1]

		if filetype == "html" {
			w.WriteString("," + filetype + ",,,,0")
		} else if filetype != "" {
			w.WriteString("," + filetype)
		} else {
			w.WriteString("")
		}
		w.Flush()
	}
	return filetype
}

// Get the file size
func getFileSize(url string, w *bufio.Writer) float64 {
	var downloadSize float64

	resp, err := http.Head(url)
	if err != nil {
		fmt.Println(err)
		erro := fmt.Sprintf("%s", err)
		w.WriteString(",Erro: " + erro)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		w.WriteString(",Erro: " + resp.Status)
	} else {
		// the Header "Content-Length" will provide the total file size to download
		size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
		downloadSize = float64(size)
		downloadSize = math.Floor(downloadSize/1024/1024*100) / 100
		downloadSizeString := fmt.Sprintf("%v", downloadSize)
		w.WriteString("," + downloadSizeString)
		w.Flush()
	}
	return downloadSize
}

// Get object field data
func getObject(urlEdm string, w *bufio.Writer) float64 {

	var (
		b, e1, e2, e3  int
		x, urlFileSize string
		urlFileS       bool
		fileSize       float64
	)

	response, err := http.Get(urlEdm)

	if err != nil {
		errRes := fmt.Sprintf("%s", err)
		w.WriteString(errRes)
	}

	contents, err := ioutil.ReadAll(response.Body)

	if err != nil {
		errCon := fmt.Sprintf("%s", err)
		w.WriteString(errCon)
	} else {

		defer response.Body.Close()

		content := string(contents)
		b = strings.Index(content, "<edm:object rdf:resource=")
		e1 = strings.Index(content, "\"/><edm:hasView rdf:resource")
		e2 = strings.Index(content, "\"/><edm:isShownBy rdf:resource=")
		e3 = strings.Index(content, "\"/><edm:rights rdf:resource=")

		if b != -1 && e1 != -1 && e1 > b {
			x = content[b:e1]
			urlFileSize = fmt.Sprintf("%s", x[26:])
			urlFileS = strings.Contains(urlFileSize, "\"")
			if urlFileS == false {
				fileSize = getFileSize(urlFileSize, w)
			}
		} else if b != -1 && e2 != -1 && e2 > b {
			x = content[b:e2]
			urlFileSize = fmt.Sprintf("%s", x[26:])
			urlFileS = strings.Contains(urlFileSize, "\"")
			if urlFileS == false {
				fileSize = getFileSize(urlFileSize, w)
			}
		} else if b != -1 && e3 != -1 && e3 > b {
			x = content[b:e3]
			urlFileSize = fmt.Sprintf("%s", x[26:])
			urlFileS = strings.Contains(urlFileSize, "\"")
			if urlFileS == false {
				fileSize = getFileSize(urlFileSize, w)
			}
		} else {
			w.WriteString(", Does not have the object field")
			w.Flush()
		}
	}

	return fileSize
}

func edmType(urlEdm string) string {
	var resourceType string

	res, err := http.Get(urlEdm)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	// Check if edm\\:type exists
	doc.Find("edm\\:type").Each(func(i int, s *goquery.Selection) {
		resourceType = fmt.Sprintf("%s", s.Text())
	})

	fmt.Printf("resourceType %v", resourceType)
	return resourceType

}
