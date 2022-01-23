package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/fetch"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/mantyr/pricer"
)

func main() {
	outputDirName := fmt.Sprintf("log/%d", time.Now().Unix())
	itemDirPath := fmt.Sprintf("%s/shop_%s/item_%d", outputDirName, "shop", "item")
	os.MkdirAll(itemDirPath, os.ModePerm)

	// start()
	// fmt.Printf("launch cron")
	// c := cron.New()
	// _, err := c.AddFunc("@every 6h", start)
	// if err != nil {
	// 	fmt.Errorf(err.Error())
	// } else {
	// 	c.Run()
	// }
}

func start() {
	fmt.Printf("\nok, let's go\n")
	fmt.Printf("trying to connect to db\n")
	db, err := sql.Open("sqlite3", "pricewatcher.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Printf("receive shop items\n")
	var shopItems = make(map[Shop][]Item)
	shopItems, err = getShopItems(db)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("create directory for assets\n")
	outputDirName := fmt.Sprintf("log/%d", time.Now().Unix())
	fmt.Printf("shop items count: %d\n", len(shopItems))
	fmt.Printf("process shop items\n")
	for shop, items := range shopItems {
		process(shop, items, outputDirName, db)
	}
}

func process(shop Shop, items []Item, outputDirName string, db *sql.DB) {
	fmt.Printf("process shop items\n")
	var err error
	for itemIndex := range items {
		item := items[itemIndex]
		err = receiveAndStoreItemPrice(shop, items[itemIndex], outputDirName, db)
		if err != nil {
			log.Printf("can't parse price from: %s,\n error: %v\n", item.title, err.Error())
		}
	}
}

func receiveAndStoreItemPrice(shop Shop, item Item, dir string, db *sql.DB) error {
	fmt.Printf("receive price from shop:\n")
	fmt.Printf("shop id: %d, item id: %d, item title: %s\n", shop.id, item.id, item.title)
	itemDirPath, err, itemScreenshotPath, itemPagePath := createAssetDirs(shop, item, dir)
	if err != nil {
		fmt.Printf("can't create one of asset dir: %v\n", err)
	} else {
		pageContent, err := getPageContent(item.link)
		if err == nil {
			fmt.Printf("store page screenshot to %s\n", itemScreenshotPath)
			err = ioutil.WriteFile(itemScreenshotPath, pageContent.screenshoot, 0644)
			if err != nil {
				fmt.Printf("Can't write page screenshot on disk %v\n", err)
			}

			fmt.Printf("store page html to %s\n", itemPagePath)
			err = ioutil.WriteFile(itemPagePath, pageContent.content, 0644)
			if err != nil {
				fmt.Printf("Can't write page content on disk %v\n", err)
			}

			fmt.Printf("parse price \n")
			var parsedPrice, err = parsePrice(pageContent, shop, item)
			fmt.Printf("item: %s price: %s\n", item.title, parsedPrice)
			if err == nil {
				err = storePrice(parsedPrice, item, itemDirPath, err, db)
			} else {
				fmt.Printf("can't parse page. %v\n", err)
			}
		} else {
			fmt.Printf("can't get page content: %v\n", err)
		}
	}
	return err
}

func createAssetDirs(shop Shop, item Item, dir string) (string, error, string, string) {
	itemDirPath := fmt.Sprintf("%s/shop_%s/item_%d", dir, shop.title, item.id)
	err := os.MkdirAll(itemDirPath, os.ModePerm)
	itemScreenshotPath := itemDirPath + "/screenshot.png"
	itemPagePath := itemDirPath + "/page.html"
	return itemDirPath, err, itemScreenshotPath, itemPagePath
}

func parsePrice(pageContent PageContent, shop Shop, item Item) (string, error) {
	var parsedPrice string
	var err error
	doc, err := htmlquery.Parse(strings.NewReader(string(pageContent.content)))
	if err == nil {
		parsedPrice, err = getPrice(doc, shop, item)
	} else {
		fmt.Printf("can't parse page %v\n", err)
	}
	return parsedPrice, err
}

func storePrice(parsedPrice string, item Item, itemDirPath string, err error, db *sql.DB) error {
	price := pricer.NewPrice()
	price.SetDefaultType("UNKNOWN")
	price.Parse(parsedPrice)

	filteredPrice := price.Get()
	currency := price.GetType()
	stmt := fmt.Sprintf(
		"insert into prices (item_id, parsed_price, filtered_price, timestamp, assets_path, currency) values(%d, \"%s\", \"%s\", %d, \"%s\", \"%s\")",
		item.id,
		parsedPrice,
		filteredPrice,
		time.Now().Unix(),
		itemDirPath,
		currency,
	)

	fmt.Printf("trying to execute: %s\n", stmt)
	_, err = db.Exec(stmt)
	return err
}

func getPrice(html *html.Node, shop Shop, item Item) (string, error) {
	var err error
	var price string
	priceXPath := fmt.Sprintf("//%s", shop.priceXPath)
	priceNode, err := htmlquery.Query(html, priceXPath)
	if err == nil {
		if priceNode != nil && priceNode.FirstChild != nil {
			price = priceNode.FirstChild.Data
		} else {
			err = errors.New(fmt.Sprintf("can't parse %s", item.title))
		}
	} else {
		log.Fatal(err)
	}
	return price, err
}

func getShops(db *sql.DB) ([]Shop, error) {
	var shops []Shop
	rows, err := db.Query("select * from shops")
	if err != nil {
		return shops, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var title, titleXPath, pictureXPath, priceXPath string
		err = rows.Scan(&id, &title, &titleXPath, &pictureXPath, &priceXPath)
		if err != nil {
			return shops, err
		}
		shops = append(shops, Shop{id, title, titleXPath, pictureXPath, priceXPath})
	}
	return shops, err
}

func getItemsByShop(db *sql.DB, shopId int) ([]Item, error) {
	var items []Item
	stmt, err := db.Query("select * from items where shop_id = $1", shopId)
	if err != nil {
		return items, err
	}
	defer stmt.Close()
	for stmt.Next() {
		var id, shopId int
		var title, link string
		err = stmt.Scan(&id, &title, &link, &shopId)
		if err != nil {
			return items, err
		}
		items = append(items, Item{id, title, link, shopId})
	}
	return items, err
}

func getShopItems(db *sql.DB) (map[Shop][]Item, error) {
	var shopItems = make(map[Shop][]Item)
	shops, err := getShops(db)
	if err != nil {
		return shopItems, err
	}
	for _, shop := range shops {
		var items []Item
		items, err = getItemsByShop(db, shop.id)
		if err != nil {
			return shopItems, err
		}
		shopItems[shop] = items
	}
	return shopItems, err
}

func getPageContent(url string) (PageContent, error) {
	fmt.Printf("get page content: %s\n", url)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer("use your proxy server"),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	lctx, lcancel := context.WithCancel(ctx)
	chromedp.ListenTarget(lctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				_ = chromedp.Run(ctx, fetch.ContinueRequest(ev.RequestID))
			}()
		case *fetch.EventAuthRequired:
			if ev.AuthChallenge.Source == fetch.AuthChallengeSourceProxy {
				go func() {
					_ = chromedp.Run(ctx,
						fetch.ContinueWithAuth(ev.RequestID, &fetch.AuthChallengeResponse{
							Response: fetch.AuthChallengeResponseResponseProvideCredentials,
							Username: "proxy user name",
							Password: "prixy user password",
						}),
						// Chrome will remember the credential for the current instance,
						// so we can disable the fetch domain once credential is provided.
						// Please file an issue if Chrome does not work in this way.
						fetch.Disable(),
					)
					// and cancel the event handler too.
					lcancel()
				}()
			}
		}
	})

	var pageContent string
	var screenshot []byte

	err := chromedp.Run(
		ctx,
		fetch.Enable().WithHandleAuthRequests(true),
		chromedp.Emulate(device.IPhone11),
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second),
		chromedp.MouseEvent(input.MouseMoved, 15, 20),
		chromedp.CaptureScreenshot(&screenshot),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			pageContent, err =
				dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	)
	return PageContent{[]byte(pageContent), screenshot}, err
}

// Data

type PageContent struct {
	content     []byte
	screenshoot []byte
}

// Database entities

type Item struct {
	id      int
	title   string
	link    string
	shop_id int
}

type Shop struct {
	id           int
	title        string
	titleXPath   string
	pictureXPath string
	priceXPath   string
}
