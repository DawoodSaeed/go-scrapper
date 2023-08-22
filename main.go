package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/html"
	"gopkg.in/olivere/elastic.v5"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Car struct {
	Name     string
	Price    string
	Rating   string
	Location string
	Year     int
	Mileage  string
	Type     string
	CC       string
	GearBox  string
}

func main() {

	client, elasticConnectionErr := elastic.NewClient(elastic.SetURL("http://localhost:9200"), elastic.SetSniff(false))
	errorHandling(elasticConnectionErr)

	baseUrl := "https://www.pakwheels.com/used-cars/search/-/"

	//Making a http client to pass the SSL certificate
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{
		TLSClientConfig: config,
	}
	netClient := &http.Client{
		Transport: transport,
	}

	//Making a request to the URL
	response, httpError := netClient.Get(baseUrl)

	//Checking if there is a http Error
	if httpError != nil {
		fmt.Println("Error: ", httpError)
		os.Exit(1)
	}

	//Parsing the HTML
	doc, htmlParseError := html.Parse(response.Body)
	errorHandling(htmlParseError)
	divs := getElementByClass(doc, "search-list")
	for _, div := range divs {
		var car Car
		// Getting the car name....
		carNames := getElementByClass(div, "car-name")
		// Getting the car Price
		prices := getElementByClass(div, "price-details")

		//getting the car rating
		carRatings := getElementByClass(div, "auction-rating")

		// Getting the car location
		carLocations := getElementByClass(div, "search-vehicle-info")
		if len(carNames) <= 0 || len(prices) <= 0 || len(carLocations) <= 0 {
			continue
		}
		var carRating string
		if len(carRatings) > 0 {
			carRating = getElementText(carRatings[0])
			car.Rating = carRating
		} else {
			car.Rating = "0"
		}

		price := getElementText(prices[0])
		carName := getElementText(carNames[0])
		carLocation := getElementText(carLocations[0])

		car.Name = carName
		car.Location = carLocation
		car.Price = price

		fmt.Println("Car name: ", carName, "Price: ", price, "Car Location: ", carLocation, " car rating: ", carRating)

		//	Get ul and then the lis
		lis := getElementByClass(div, "search-vehicle-info-2")
		for index, li := range getElementsByTagName(lis[0], "li") {
			info := getElementText(li)
			fmt.Println(index, info)

			switch index {
			case 0:
				car.Year, _ = strconv.Atoi(info)
				break
			case 1:
				car.Mileage = info
				break
			case 2:
				car.Type = info
				break
			case 3:
				car.CC = info
				break
			case 4:
				car.GearBox = info
			}

		}
		_, insertionError := client.Index().
			Index("pakwheels").
			Type("car").
			BodyJson(car).
			Do(context.Background())

		errorHandling(insertionError)
	}
	//Closing the IO stream
	response.Body.Close()
}

func errorHandling(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

func getElementByClass(node *html.Node, class string) []*html.Node {
	var result []*html.Node

	//If its a element Node
	if node.Type == html.ElementNode {
		for _, attr := range node.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, class) {
				result = append(result, node)
			}
		}

	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, getElementByClass(c, class)...)
	}

	return result
}

// For printing the HTML

func printHTMLNode(node *html.Node, indent string) {
	fmt.Printf("%s<%s", indent, node.Data)

	// Print attributes
	for _, attr := range node.Attr {
		fmt.Printf(" %s=\"%s\"", attr.Key, attr.Val)
	}

	fmt.Println(">")

	// Print child nodes
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			printHTMLNode(c, indent+"  ")
		} else if c.Type == html.TextNode {
			fmt.Printf("%s  %s\n", indent+"  ", c.Data)
		}
	}

	fmt.Printf("%s</%s>\n", indent, node.Data)
}

// Helper function to get the text content of an element
func getElementText(n *html.Node) string {
	var text string

	if n.Type == html.TextNode {
		text = n.Data
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += getElementText(c)
	}

	return strings.TrimSpace(text)
}

func getElementsByTagName(node *html.Node, tagName string) []*html.Node {
	var result []*html.Node

	if node.Type == html.ElementNode && node.Data == tagName {
		result = append(result, node)
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, getElementsByTagName(c, tagName)...)
	}

	return result
}
