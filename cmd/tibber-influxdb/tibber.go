package main

import (
	"context"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
)

type TibberConfig struct {
	Token    string
	Endpoint string
}

type response struct {
	Viewer struct {
		Name  string `json:"name"`
		Homes []struct {
			ID       string `json:"id"`
			TimeZone string `json:"timeZone"`
			Address  struct {
				Address1   string `json:"address1"`
				Address2   string `json:"address2"`
				Address3   string `json:"address3"`
				City       string `json:"city"`
				PostalCode string `json:"postalCode"`
				Country    string `json:"country"`
				Latitude   string `json:"latitude"`
				Longitude  string `json:"longitude"`
			} `json:"address"`
			Owner struct {
				FirstName   string `json:"firstName"`
				LastName    string `json:"lastName"`
				ContactInfo struct {
					Email  string `json:"email"`
					Mobile string `json:"mobile"`
				} `json:"contactInfo"`
			} `json:"owner"`
			Consumption struct {
				Nodes []struct {
					From            time.Time `json:"from"`
					To              time.Time `json:"to"`
					Cost            float64   `json:"cost"`
					UnitPrice       float64   `json:"unitPrice"`
					UnitPriceVAT    float64   `json:"unitPriceVAT"`
					Currency        string    `json:"currency"`
					Consumption     float64   `json:"consumption"`
					ConsumptionUnit string    `json:"consumptionUnit"`
				} `json:"nodes"`
			} `json:"consumption"`
			CurrentSubscription struct {
				PriceRating struct {
					Hourly struct {
						MinTotal  float64 `json:"minTotal"`
						MaxTotal  float64 `json:"maxTotal"`
						Currency  string  `json:"currency"`
						MinEnergy float64 `json:"minEnergy"`
						MaxEnergy float64 `json:"maxEnergy"`
						Entries   []struct {
							Difference float64   `json:"difference"`
							Tax        float64   `json:"tax"`
							Energy     float64   `json:"energy"`
							Total      float64   `json:"total"`
							Time       time.Time `json:"time"`
							Level      string    `json:"level"`
						} `json:"entries"`
					} `json:"hourly"`
				} `json:"priceRating"`
			} `json:"currentSubscription"`
		} `json:"homes"`
	} `json:"viewer"`
}

func scrape(ctx context.Context, token string, client *graphql.Client) (*response, error) {
	req := graphql.NewRequest(`
{
  viewer {
    name
    homes {
      id
      timeZone
      address {
        address1
		    address2
		    address3
        city
        postalCode
		    country
		    latitude
		    longitude
      }
      owner {
        firstName
        lastName
        contactInfo {
          email
          mobile
        }
      }
      consumption (resolution: HOURLY, last: 48){
         nodes {
          from
          to
          cost
          unitPrice
          unitPriceVAT
          currency
          consumption
          consumptionUnit
        }
      }
      currentSubscription{
        priceRating {
          hourly {
            minTotal
            maxTotal
            currency
            minEnergy
            maxEnergy
            entries {
              difference
              tax
              energy
              total
              time
              level
            }
          }
        }
      }
    }
  }
}

`)
	req.Header.Set("Authorization", token)
	response := &response{}
	err := client.Run(ctx, req, response)
	if err != nil {
		return nil, fmt.Errorf("failed to get data from tibber: %s", err)
	}
	return response, nil
}
