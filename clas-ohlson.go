package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ClasOhlson struct {
	StoreID string // 200
}

type ClasOhlsonItem struct {
	ID   string
	Text string
}

func (c *ClasOhlson) Search(text string) ([]ClasOhlsonItem, error) {
	resp, err := http.Get("https://www.clasohlson.com/no/search/getSearchResults?text=" + url.QueryEscape(text))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		GtmValues struct {
			ProductList []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"productList"`
		} `json:"gtmValues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]ClasOhlsonItem, len(result.GtmValues.ProductList))
	for i, p := range result.GtmValues.ProductList {
		items[i] = ClasOhlsonItem{ID: p.ID, Text: p.Name}
	}
	return items, nil
}

func (c *ClasOhlson) Availability(item ClasOhlsonItem) (int, error) {
	req, _ := http.NewRequest("GET", "https://www.clasohlson.com/no/cocheckout/getCartDataOnReload?variantProductCode="+item.ID, nil)
	req.Header.Set("Cookie", "COStoreCookie="+c.StoreID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		StoreStockList struct {
			StockData []struct {
				StoreID    string `json:"storeId"`
				StoreStock int    `json:"storeStock"`
			} `json:"stockData"`
		} `json:"storeStockList"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	for _, s := range result.StoreStockList.StockData {
		if s.StoreID == c.StoreID {
			return s.StoreStock, nil
		}
	}
	return 0, fmt.Errorf("store %s not found", c.StoreID)
}

// for mr llm
// func TestFindAvailability(t *testing.T) {
// 	queryA := "https://www.clasohlson.com/no/search/getSearchResults?text=skopose"
// 	queryB := "https://www.clasohlson.com/no/cocheckout/getCartDataOnReload?variantProductCode=445689000"

// }

// http  'https://www.clasohlson.com/no/cocheckout/getCartDataOnReload?variantProductCode=445689000' Cookie:COStoreCookie=200

var exampleResponse = `
content-type: application/json;charset=UTF-8
expires: 0
pragma: no-cache
server: Apache
set-cookie: JSESSIONID=910659D6634C395101C0C4811C0FA7FF.prodccom10; Path=/no; Secure; HttpOnly, apptus.customerKey=36ea2438-bf7d-4b8a-9499-f41948ff1a25; Max-Age=315360000; Expires=Tue, 29-Jan-2036 18:41:27 GMT; Path=/; Secure; HttpOnly, apptus.sessionKey=ac1d14aa-a80d-4345-839d-76658175354e; Path=/; Secure; HttpOnly, SessionSiteView=B2C; Path=/no; Secure, pdpCacheCookie=B2C-A-true-no; Max-Age=86400; Expires=Sun, 01-Feb-2026 18:41:27 GMT; Path=/no; Secure; HttpOnly
transfer-encoding: chunked
vary: Accept-Encoding
via: 1.1 varnish, 1.1 varnish, 1.1 varnish
x-content-type-options: nosniff
x-frame-options: SAMEORIGIN
x-xss-protection: 1; mode=block

{
    "categoryResponseDTOs": {
        "categories": [
            {
                "code": "1036",
                "count": 4,
                "name": "Fritid",
                "preSelected": false,
                "subCategories": [
                    {
                        "code": "1911",
                        "count": 1,
                        "name": "Hobby- og kunstmateriell",
                        "preSelected": false,
                        "subCategories": [
                            {
                                "code": "2979",
                                "count": 1,
                                "name": "Håndarbeid og hobbysett",
                                "preSelected": false,
                                "subCategories": [],
                                "url": "/no/Fritid/Hobby--og-kunstmateriell/Håndarbeid-og-hobbysett/c/2979"
                            }
                        ],
                        "url": "/no/Fritid/Hobby--og-kunstmateriell/c/1911"
                    },
                    {
                        "code": "1908",
                        "count": 1,
                        "name": "Leker og spill",
                        "preSelected": false,
                        "subCategories": [
                            {
                                "code": "2265",
                                "count": 1,
                                "name": "Hobbysett til barn",
                                "preSelected": false,
                                "subCategories": [],
                                "url": "/no/Fritid/Leker-og-spill/Hobbysett-til-barn/c/2265"
                            }
                        ],
                        "url": "/no/Fritid/Leker-og-spill/c/1908"
                    },
                    {
                        "code": "1546",
                        "count": 3,
                        "name": "Reise",
                        "preSelected": false,
                        "subCategories": [
                            {
                                "code": "2776",
                                "count": 3,
                                "name": "Reiseeffekter",
                                "preSelected": false,
                                "subCategories": [],
                                "url": "/no/Fritid/Reise/Reiseeffekter/c/2776"
                            }
                        ],
                        "url": "/no/Fritid/Reise/c/1546"
                    }
                ],
                "url": "/no/Fritid/c/1036"
            },
            {
                "code": "1035",
                "count": 4,
                "name": "Hjem",
                "preSelected": false,
                "subCategories": [
                    {
                        "code": "1844",
                        "count": 3,
                        "name": "Oppbevaring og organisering",
                        "preSelected": false,
                        "subCategories": [
                            {
                                "code": "2487",
                                "count": 3,
                                "name": "Sko-oppbevaring",
                                "preSelected": false,
                                "subCategories": [],
                                "url": "/no/Hjem/Oppbevaring-og-organisering/Sko-oppbevaring/c/2487"
                            }
                        ],
                        "url": "/no/Hjem/Oppbevaring-og-organisering/c/1844"
                    },
                    {
                        "code": "1447",
                        "count": 1,
                        "name": "Vaskerom",
                        "preSelected": false,
                        "subCategories": [
                            {
                                "code": "2481",
                                "count": 1,
                                "name": "Skotilbehør",
                                "preSelected": false,
                                "subCategories": [],
                                "url": "/no/Hjem/Vaskerom/Skotilbehør/c/2481"
                            }
                        ],
                        "url": "/no/Hjem/Vaskerom/c/1447"
                    }
                ],
                "url": "/no/Hjem/c/1035"
            }
        ],
        "contentPerPageCount": 0,
        "contentTotalCount": 0,
        "fixedFacets": [
            {
                "header": "Pris",
                "items": [
                    {
                        "preSelected": false,
                        "value": "39.9"
                    },
                    {
                        "preSelected": false,
                        "value": "199.9"
                    }
                ],
                "key": "sortingPrice"
            },
            {
                "header": "Lagerstatus",
                "items": [
                    {
                        "preSelected": false,
                        "value": "Vis kun produkter som er på nettlager"
                    }
                ],
                "key": "stockStatus"
            },
            {
                "key": "saleCampaignStatus"
            },
            {
                "key": "sparePartStatus"
            },
            {
                "header": "Merke",
                "items": [
                    {
                        "preSelected": false,
                        "value": "CLAS OHLSON"
                    },
                    {
                        "preSelected": false,
                        "value": "CREATIV COMPANY"
                    }
                ],
                "key": "brand"
            }
        ],
        "noResultFound": false,
        "noResultFoundForHystrix": false,
        "perPageCount": 18,
        "products": [
            {
                "addToCart": true,
                "apptusTicket": "Oy9zZWFyY2gvcmVzdWx0L3NlYXJjaC1yZXN1bHQvc2VhcmNoLWhpdHM7Iztwcm9kdWN0X2tleTtQcjQ0NTY4OTAwMF9OTzs0NC01Njg5X05PO09CSkVDVElWRSQ7Tk9ORTpOT05FOzE2Ow",
                "brand": "CLAS OHLSON",
                "brandImage": "/medias/sys_master/h5a/h16/68677769199646.png",
                "campaignStatus": false,
                "clasOfficePrice": 0.0,
                "clubclasPrice": 0.0,
                "code": "44-5689",
                "compUnit": "stk",
                "comparisonPrice": 19.95,
                "corpCustomerLoggedIn": false,
                "currentPrice": 39.9,
                "dangerousGood": false,
                "description": "Hold skoene dine friske og luktfrie med sedertre i poser. Sedertreet absorberer ....",
                "displayBrandeRoles": [],
                "formattedClubclasPrice": "0,00",
                "formattedCompPrice": "19,95",
                "formattedCurrentPrice": "39,90",
                "formattedOldPrice": "0,00",
                "gridViewImage": "/medias/sys_master/h30/h05/68738507210782.png",
                "grouped": true,
                "gtmId": "445689000",
                "highMargin": false,
                "inStock": true,
                "innovation": false,
                "internetOnly": false,
                "listForGTM": "search",
                "mainCategoryId": "2481",
                "mainCategoryName": "Skotilbehør",
                "mainCategoryPath": "/Hjem/Vaskerom/Skotilbehør/c/2481",
                "multiVariant": "false",
                "name": "Skopose med sedertre, 2-pakning",
                "newArrival": false,
                "oldPrice": 0.0,
                "onSale": false,
                "path": "/Skopose-med-sedertre,-2-pakning/p/44-5689",
                "productStatus": "ACTIVE",
                "productType": "PRODUCT",
                "r12": false,
                "rating": "4-0",
                "reviews": 148,
                "sortingPriceCorp": 0.0,
                "sparePartStatus": false,
                "specialCampaign": false,
                "teasers": false,
                "thumbnailImage": "/medias/sys_master/h6f/h1a/68733911891998.png",
                "ugPromoRestriction": false,
                "url": "/Skopose-med-sedertre,-2-pakning/p/44-5689",
                "variantStatus": "445689000"
            },
            {
                "addToCart": true,
                "apptusTicket": "Oy9zZWFyY2gvcmVzdWx0L3NlYXJjaC1yZXN1bHQvc2VhcmNoLWhpdHM7Iztwcm9kdWN0X2tleTtQcjQ0NDk5NTAwMV9OTzs0NC00OTk1LTFfTk87T0JKRUNUSVZFJDtOT05FOk5PTkU7MTY7",
                "brand": "CREATIV COMPANY",
                "campaignStatus": false,
                "clasOfficePrice": 0.0,
                "clubclasPrice": 0.0,
                "code": "44-4995-1",
                "comparisonPrice": 0.0,
                "corpCustomerLoggedIn": false,
                "currentPrice": 99.9,
                "dangerousGood": false,
                "description": "Design din egen skopose – tøypose med trekksnor i lys, ufarget bomull. Creativ C....",
                "displayBrandeRoles": [],
                "formattedClubclasPrice": "0,00",
                "formattedCurrentPrice": "99,90",
                "formattedOldPrice": "0,00",
                "gridViewImage": "/medias/sys_master/h92/h95/68737057980446.png",
                "grouped": true,
                "gtmId": "444995001",
                "highMargin": false,
                "inStock": true,
                "innovation": false,
                "internetOnly": false,
                "listForGTM": "search",
                "mainCategoryId": "2979",
                "mainCategoryName": "Håndarbeid og hobbysett",
                "mainCategoryPath": "/Fritid/Hobby--og-kunstmateriell/Håndarbeid-og-hobbysett/c/2979",
                "multiVariant": "false",
                "name": "Creativ Company skopose bomull 37x41 cm 3-pakning",
                "newArrival": false,
                "oldPrice": 0.0,
                "onSale": false,
                "path": "/Creativ-Company-skopose-bomull-37x41-cm-3-pakning/p/44-4995-1",
                "productStatus": "ACTIVE",
                "productType": "PRODUCT",
                "r12": false,
                "rating": "0-0",
                "reviews": 0,
                "sortingPriceCorp": 0.0,
                "sparePartStatus": false,
                "specialCampaign": false,
                "teasers": false,
                "thumbnailImage": "/medias/sys_master/h02/h96/68688381640734.png",
                "ugPromoRestriction": false,
                "url": "/Creativ-Company-skopose-bomull-37x41-cm-3-pakning/p/44-4995-1",
                "variantStatus": "444995001"
            },
            {
                "addToCart": true,
                "apptusTicket": "Oy9zZWFyY2gvcmVzdWx0L3NlYXJjaC1yZXN1bHQvc2VhcmNoLWhpdHM7Iztwcm9kdWN0X2tleTtQcjMxNzY1MDAwMF9OTzszMS03NjUwX05PO09CSkVDVElWRSQ7Tk9ORTpOT05FOzE2Ow",
                "brand": "CLAS OHLSON",
                "brandImage": "/medias/sys_master/h5a/h16/68677769199646.png",
                "campaignStatus": false,
                "clasOfficePrice": 0.0,
                "clubclasPrice": 0.0,
                "code": "31-7650",
                "comparisonPrice": 0.0,
                "corpCustomerLoggedIn": false,
                "currentPrice": 199.9,
                "dangerousGood": false,
                "description": "Pakk systematisk – del opp klærne i lett tilgjengelige kuber i kofferten. Reiseo....",
                "displayBrandeRoles": [],
                "formattedClubclasPrice": "0,00",
                "formattedCurrentPrice": "199,90",
                "formattedOldPrice": "0,00",
                "gridViewImage": "/medias/sys_master/h1e/hd1/68737659699230.png",
                "grouped": true,
                "gtmId": "317650000",
                "highMargin": false,
                "inStock": true,
                "innovation": false,
                "internetOnly": false,
                "listForGTM": "search",
                "mainCategoryId": "2776",
                "mainCategoryName": "Reiseeffekter",
                "mainCategoryPath": "/Fritid/Reise/Reiseeffekter/c/2776",
                "multiVariant": "false",
                "name": "Pakkekuber og skoposer for reiser, svarte, 7-pakning",
                "newArrival": false,
                "oldPrice": 0.0,
                "onSale": false,
                "path": "/Pakkekuber-og-skoposer-for-reiser,-svarte,-7-pakning/p/31-7650",
                "productStatus": "ACTIVE",
                "productType": "PRODUCT",
                "r12": false,
                "rating": "4-5",
                "reviews": 58,
                "sortingPriceCorp": 0.0,
                "sparePartStatus": false,
                "specialCampaign": false,
                "teasers": false,
                "thumbnailImage": "/medias/sys_master/h07/h2c/68735152619550.png",
                "ugPromoRestriction": false,
                "url": "/Pakkekuber-og-skoposer-for-reiser,-svarte,-7-pakning/p/31-7650",
                "variantStatus": "317650000"
            },
            {
                "addToCart": true,
                "apptusTicket": "Oy9zZWFyY2gvcmVzdWx0L3NlYXJjaC1yZXN1bHQvc2VhcmNoLWhpdHM7Iztwcm9kdWN0X2tleTszMTc2NDgwMDBfTk87MzEtNzY0OF9OTztPQkpFQ1RJVkUkO05PTkU6Tk9ORTsxNjs",
                "brand": "CLAS OHLSON",
                "brandImage": "/medias/sys_master/h5a/h16/68677769199646.png",
                "campaignStatus": false,
                "clasOfficePrice": 0.0,
                "clubclasPrice": 0.0,
                "code": "31-7648",
                "comparisonPrice": 0.0,
                "corpCustomerLoggedIn": false,
                "currentPrice": 119.9,
                "dangerousGood": false,
                "description": "Bærbar skooppbevaring for opptil 3 par sko – ....",
                "displayBrandeRoles": [],
                "feature": "Utførelse: Large - 26x20x36 cm ",
                "featureMobile": "Utførelse: Large - 26x20x36 cm...",
                "formattedClubclasPrice": "0,00",
                "formattedCurrentPrice": "119,90",
                "formattedOldPrice": "0,00",
                "gridViewImage": "/medias/sys_master/h72/hf8/68737659240478.png",
                "grouped": false,
                "gtmId": "317648000",
                "highMargin": false,
                "inStock": true,
                "innovation": false,
                "internetOnly": false,
                "listForGTM": "search",
                "mainCategoryId": "2487",
                "mainCategoryName": "Sko-oppbevaring",
                "mainCategoryPath": "/Hjem/Oppbevaring-og-organisering/Sko-oppbevaring/c/2487",
                "multiVariant": "true",
                "name": "Skokube, skopose til reise, for 3 par sko, grå",
                "newArrival": false,
                "oldPrice": 0.0,
                "onSale": false,
                "path": "/Skokube,-skopose-til-reise,-for-3-par-sko,-gra/p/31-7648",
                "productStatus": "ACTIVE",
                "productType": "PRODUCT",
                "r12": false,
                "rating": "4-5",
                "reviews": 26,
                "sortingPriceCorp": 0.0,
                "sparePartStatus": false,
                "specialCampaign": false,
                "teasers": false,
                "thumbnailImage": "/medias/sys_master/hed/h57/68735152160798.png",
                "ugPromoRestriction": false,
                "url": "/Skokube,-skopose-til-reise,-for-3-par-sko,-gra/p/31-7648",
                "varMap": {
                    "Utførelse": " Large - 26x20x36 cm "
                },
                "variantStatus": "product"
            },
            {
                "addToCart": true,
                "apptusTicket": "Oy9zZWFyY2gvcmVzdWx0L3NlYXJjaC1yZXN1bHQvc2VhcmNoLWhpdHM7Iztwcm9kdWN0X2tleTszMTc2NDkwMDBfTk87MzEtNzY0OV9OTztPQkpFQ1RJVkUkO05PTkU6Tk9ORTsxNjs",
                "brand": "CLAS OHLSON",
                "brandImage": "/medias/sys_master/h5a/h16/68677769199646.png",
                "campaignStatus": false,
                "clasOfficePrice": 0.0,
                "clubclasPrice": 0.0,
                "code": "31-7649",
                "comparisonPrice": 0.0,
                "corpCustomerLoggedIn": false,
                "currentPrice": 99.9,
                "dangerousGood": false,
                "description": "Bærbar skooppbevaring for opptil 3 par sko – ....",
                "displayBrandeRoles": [],
                "feature": "Utførelse: Small - 22x15x33 cm ",
                "featureMobile": "Utførelse: Small - 22x15x33 cm...",
                "formattedClubclasPrice": "0,00",
                "formattedCurrentPrice": "99,90",
                "formattedOldPrice": "0,00",
                "gridViewImage": "/medias/sys_master/h37/he8/68737659404318.png",
                "grouped": false,
                "gtmId": "317649000",
                "highMargin": false,
                "inStock": true,
                "innovation": false,
                "internetOnly": false,
                "listForGTM": "search",
                "mainCategoryId": "2487",
                "mainCategoryName": "Sko-oppbevaring",
                "mainCategoryPath": "/Hjem/Oppbevaring-og-organisering/Sko-oppbevaring/c/2487",
                "multiVariant": "true",
                "name": "Skokube, skopose til reise, for 3 par sko, grå",
                "newArrival": false,
                "oldPrice": 0.0,
                "onSale": false,
                "path": "/Skokube,-skopose-til-reise,-for-3-par-sko,-gra/p/31-7649",
                "productStatus": "ACTIVE",
                "productType": "PRODUCT",
                "r12": false,
                "rating": "4-5",
                "reviews": 26,
                "sortingPriceCorp": 0.0,
                "sparePartStatus": false,
                "specialCampaign": false,
                "teasers": false,
                "thumbnailImage": "/medias/sys_master/h28/h68/68735152324638.png",
                "ugPromoRestriction": false,
                "url": "/Skokube,-skopose-til-reise,-for-3-par-sko,-gra/p/31-7649",
                "varMap": {
                    "Utførelse": " Small - 22x15x33 cm "
                },
                "variantStatus": "product"
            }
        ],
        "relevantFacets": [],
        "storePerPageCount": 0,
        "storeTotalCount": 0,
        "totalCount": 5
    },
    "dYMListAvailable": false,
    "didYouMeanSearchText": "Viser resultat for <strong></strong> i stedet",
    "gtmValues": {
        "productList": [
            {
                "Brand": "CLAS OHLSON",
                "category": "Hjem/Vaskerom/Skotilbehør",
                "category_id": "2481",
                "id": "445689000",
                "list": "search",
                "name": "Skopose med sedertre  2-pakning",
                "position": "1",
                "price": "39.90",
                "variant": ""
            },
            {
                "Brand": "CREATIV COMPANY",
                "category": "Fritid/Hobby--og-kunstmateriell/Håndarbeid-og-hobbysett",
                "category_id": "2979",
                "id": "444995001",
                "list": "search",
                "name": "Creativ Company skopose bomull 37x41 cm 3-pakning",
                "position": "2",
                "price": "99.90",
                "variant": ""
            },
            {
                "Brand": "CLAS OHLSON",
                "category": "Fritid/Reise/Reiseeffekter",
                "category_id": "2776",
                "id": "317650000",
                "list": "search",
                "name": "Pakkekuber og skoposer for reiser  svarte  7-pakning",
                "position": "3",
                "price": "199.90",
                "variant": ""
            },
            {
                "Brand": "CLAS OHLSON",
                "category": "Hjem/Oppbevaring-og-organisering/Sko-oppbevaring",
                "category_id": "2487",
                "id": "317648000",
                "list": "search",
                "name": "Skokube  skopose til reise  for 3 par sko  grå",
                "position": "4",
                "price": "119.90",
                "variant": "Utførelse: Large - 26x20x36 cm "
            },
            {
                "Brand": "CLAS OHLSON",
                "category": "Hjem/Oppbevaring-og-organisering/Sko-oppbevaring",
                "category_id": "2487",
                "id": "317649000",
                "list": "search",
                "name": "Skokube  skopose til reise  for 3 par sko  grå",
                "position": "5",
                "price": "99.90",
                "variant": "Utførelse: Small - 22x15x33 cm "
            }
        ]
    },
    "noResultFound": false,
    "noResultFoundForHystrix": false,
    "redirectTrue": false
}
`

var exampleStoreResult = `{
    "CSRFToken": "643e0968-593f-4c8f-92a1-8729e1ae56fa",
    "cartPriceString": null,
    "count": 0,
    "customerKey": "5833f25c-4cd4-4b32-93e0-4fd09316d6d3",
    "sessionKey": "50279a03-aba9-4c88-be3f-2412fc9337b9",
    "storeStockList": {
        "showMoreButton": false,
        "showMoreLabel": null,
        "stockData": [
            {
                "shelfData": [
                    {
                        "areaName": "Hjem",
                        "shelfNumbers": "136",
                        "shelfTypeName": "Butikkhylle",
                        "storagePlaceIsBehind": false
                    },
                    {
                        "areaName": "Kasseområde",
                        "shelfNumbers": "S7",
                        "shelfTypeName": "Kurv/Styrtkurv",
                        "storagePlaceIsBehind": false
                    },
                    {
                        "areaName": "Hjem",
                        "shelfNumbers": "137",
                        "shelfTypeName": "Butikkhylle",
                        "storagePlaceIsBehind": false
                    }
                ],
                "shelfNumberLabel": "Hyllenummer",
                "showErrorMessage": false,
                "stockstatus": "inStock",
                "storeCity": "Oslo",
                "storeId": "200",
                "storeLink": "/no/store-finder?q=Oslo, CC Vest&latitude=59.917918&longitude
=10.635669&source=pdp",
                "storeLocation": " CC Vest",
                "storeName": "Oslo, CC Vest",
                "storeStatusLabel": "På lager <strong>(42 stk)</strong>, henteklar innen 30
 min",
                "storeStock": 42,
                "storeStockErrorMessage": null
            }
        ]
    }
}`
