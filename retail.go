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

type RetailScraper struct {
	StartURL string
	browser  *browser.Browser

	categories map[int]string
	products   []Product

	categoryIDExpr *regexp.Regexp
	productIDExpr  *regexp.Regexp
}

func (s *RetailScraper) Init() error {
	s.browser = surf.NewBrowser()
	err := s.browser.Open(s.StartURL)
	if err != nil {
		return err
	}
	s.categories = make(map[int]string)
	s.productIDExpr = regexp.MustCompile(`\/products\/(\d+)-`)
	s.categoryIDExpr = regexp.MustCompile(`\/category\/(\d+)-`)
	return nil
}

func (s *RetailScraper) navigate(pagerElem *goquery.Selection) (href string, keepBrowsing bool) {
	pagerElem.Find("div *").Each(func(_ int, e *goquery.Selection) {
		href, _ = e.Attr("href")
		if strings.Contains(e.Text(), "Siguiente") {
			keepBrowsing = true
			return
		}
	})
	return href, keepBrowsing
}

func (s *RetailScraper) getProductID(url string) (int, error) {
	matches := s.productIDExpr.FindAllStringSubmatch(url, 10)
	if len(matches) != 1 {
		return -1, errors.New("Couldn't match product ID")
	}
	idStr := matches[0][1]
	n, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, err
	}
	return n, nil
}

func (s *RetailScraper) getCategoryID(url string) (int, error) {
	matches := s.categoryIDExpr.FindAllStringSubmatch(url, 10)
	if len(matches) != 1 {
		return -1, errors.New("Couldn't match product ID")
	}
	idStr := matches[0][1]
	n, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, err
	}
	return n, nil
}

func (s *RetailScraper) getProductPrice(priceElem *goquery.Selection) (int, error) {
	text := priceElem.Text()
	if text == "" {
		return -1, errors.New("Empty price element text")
	}
	price := strings.Replace(text, ".", "", -1)
	price = strings.Replace(price, " ", "", -1)
	return strconv.Atoi(price)
}

func (s *RetailScraper) Fetch(productFn func(*Product)) {
	s.browser.Find("a").Each(func(_ int, catElements *goquery.Selection) {
		catURL, _ := catElements.Attr("href")
		if !strings.Contains(catURL, "/category/") {
			return
		}
		s.browser.Open(catURL)
		catID, err := s.getCategoryID(catURL)
		if err != nil {
			log.Error("Ignorando categoría (no se pudo obtener el ID)")
			return
		}
		s.categories[catID] = catURL
		for {
			pagerElem := s.browser.Find(".product-pager-box")
			s.browser.Find(".product-item").Each(func(_ int, e *goquery.Selection) {
				priceElem := e.Find(".price-label")
				titleElem := e.Find(".product-title a")
				linkElem := e.Find(".product-title-link")
				if priceElem.Length() == 0 || titleElem.Length() == 0 || linkElem.Length() == 0 {
					log.Error("Ignorando producto (elementos no encontrados)")
					return
				}
				title := titleElem.Text()
				link, exists := linkElem.Attr("href")
				if !exists {
					log.Error("Ignorando producto (no existe href en .product-title-link")
					return
				}
				productID, err := s.getProductID(link)
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
					CategoryID:  catID,
					CategoryURL: catURL,
				}
				productFn(p)
			})
			nextHref, keepBrowsing := s.navigate(pagerElem)
			if !keepBrowsing {
				break
			}
			s.browser.Open(nextHref)
		}
	})
}
