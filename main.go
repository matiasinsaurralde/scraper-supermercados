package main

import (
	"encoding/json"
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
)

// Scraper es una interfaz para los scrapers.
type Scraper interface {
	Fetch(func(*Product))
	Init() error
}

// Product es una estructura de datos para los datos del producto.
type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Price       int    `json:"price"`
	CategoryID  int    `json:"category_id"`
	CategoryURL string `json:"category_link"`
	PerKg       bool   `json:"per_kg"`
	SKU         string `json:"sku"`
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// NewScraper inicializa un nuevo scraper.
func NewScraper(typ string) (Scraper, error) {
	switch typ {
	case "s6":
		scraper := &RetailScraper{StartURL: "http://www.superseis.com.py/default.aspx"}
		err := scraper.Init()
		if err != nil {
			return nil, err
		}
		return scraper, nil
	case "stock":

		scraper := &RetailScraper{StartURL: "http://www.stock.com.py/default.aspx"}
		err := scraper.Init()
		if err != nil {
			return nil, err
		}
		return scraper, nil
	case "casarica":
		scraper := &CasaRicaScraper{}
		err := scraper.Init()
		if err != nil {
			return nil, err
		}
		return scraper, nil
	case "arete":
		scraper := &AreteScraper{}
		err := scraper.Init()
		if err != nil {
			return nil, err
		}
		return scraper, nil
	}
	return nil, errors.New("Invalid scraper ID")
}

func main() {
	scraperID := os.Args[1]
	filename := os.Args[2]
	s, err := NewScraper(scraperID)
	if err != nil {
		panic(err)
	}
	os.Create(filename)
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0644)
	s.Fetch(func(p *Product) {
		productJSON, _ := json.Marshal(p)
		log.WithFields(log.Fields{
			"scraper": scraperID,
		}).Debug("Got", string(productJSON))
		if err != nil {
			panic(err)
		}
		file.WriteString(string(productJSON) + "\n")
	})
}
