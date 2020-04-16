package main

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	log "github.com/sirupsen/logrus"
)

const (
	AreteStartURL = "https://www.arete.com.py/"
)

type AreteScraper struct {
	browser        *browser.Browser
	productBrowser *browser.Browser
	ids            map[int]int

	skuExpr        *regexp.Regexp
	categoryIDExpr *regexp.Regexp
	productIDExpr  *regexp.Regexp
	errSKUMatch    error
}

func (s *AreteScraper) Init() error {
	s.browser = surf.NewBrowser()
	err := s.browser.Open(AreteStartURL)
	if err != nil {
		return err
	}
	s.ids = make(map[int]int)
	s.skuExpr = regexp.MustCompile(`"image": "(.*?)"`)
	s.errSKUMatch = errors.New("No se pudo obtener SKU")
	return nil
}

func (s *AreteScraper) getSKU(skuElem *goquery.Selection, p *Product) error {

	srcAttr, exist := skuElem.Attr("src")
	if !exist {
		return s.errSKUMatch
	}

	imgString := strings.TrimSpace(srcAttr)
	splittedStr := strings.Split(imgString, "/")
	lastSlice := splittedStr[len(splittedStr)-1]
	splittedStr = strings.Split(lastSlice, ".")
	skuString := splittedStr[0]

	if strings.Contains(skuString, "default") {
		p.SKU = ""
	} else {
		p.SKU = skuString
	}

	return nil
}

func (s *AreteScraper) navigate() (relHref string, keepBrowsing bool) {
	pagination := s.browser.Find(".pagination")
	if pagination.Size() > 0 {
		pagination.Find("li a").Each(func(_ int, pagElem *goquery.Selection) {
			rel, found := pagElem.Attr("rel")
			if !found {
			}
			if strings.Contains(rel, "next") {
				relHref, _ = pagElem.Attr("href")
				keepBrowsing = true
				return
			}
		})
	}
	return relHref, keepBrowsing
}

func (s *AreteScraper) getProductPrice(priceElem *goquery.Selection) (int, error) {
	text := strings.TrimSpace(priceElem.Text())
	if text == "" {
		return -1, errors.New("Empty price element text")
	}
	price := strings.Replace(text, "Gs.", "", -1)
	price = strings.Replace(price, ".", "", -1)
	price = strings.Replace(price, " ", "", -1)
	return strconv.Atoi(price)
}

func (s *AreteScraper) getProductID(idElem *goquery.Selection) (int, error) {
	productID, exists := idElem.Attr("data-id")
	if !exists {
		return -1, errors.New("Empty ID element")
	}

	return strconv.Atoi(productID)
}

func (s *AreteScraper) Fetch(productFn func(*Product)) {
	s.browser.Find("#dl-menu a").Each(func(_ int, catElem *goquery.Selection) {
		catURL, _ := catElem.Attr("href")
		if !strings.Contains(catURL, "https://") {
			return
		}
		// Ignorar promociones por ahora:
		if strings.Contains(catURL, "ofertas") {
			return
		}
		// Ignorar novedades:
		if strings.Contains(catURL, "novedades") {
			return
		}

		err := s.browser.Open(catURL)
		if err != nil {
			panic(err)
		}
		for {
			s.browser.Find(".item").Each(func(_ int, productDiv *goquery.Selection) {

				priceElem := productDiv.Find(".price-product").Not(".price-discount")
				titleElem := productDiv.Find(".desc-product a")
				linkElem := titleElem
				idElem := productDiv.Find(".buy")
				skuElem := productDiv.Find(".imgproduct img")

				if priceElem.Length() == 0 || titleElem.Length() == 0 || linkElem.Length() == 0 || idElem.Length() == 0 {
					log.Error("Ignorando producto (elementos no encontrados)")
					return
				}
				title := strings.TrimSpace(titleElem.Text())
				link, exists := linkElem.Attr("href")
				if !exists {
					log.Error("Ignorando producto (no existe href en .desc-product")
					return
				}
				productID, err := s.getProductID(idElem)
				if err != nil {
					log.Error("Ignorando producto (no se pudo obtener ID)")
					return
				}
				price, err := s.getProductPrice(priceElem)
				if err != nil {
					log.Error("Ignorando producto (no se pudo procesar el precio)")
					return
				}
				p := &Product{
					ID:          productID,
					Name:        title,
					URL:         link,
					Price:       price,
					CategoryID:  0,
					CategoryURL: catURL,
				}
				err = s.getSKU(skuElem, p)
				if err != nil {
					log.Error(err)
				}
				productFn(p)
			})
			nextHref, keepBrowsing := s.navigate()
			if !keepBrowsing {
				break
			}
			s.browser.Open(nextHref)
		}
	})
}
