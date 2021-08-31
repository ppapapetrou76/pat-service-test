package geoip

import (
	"context"
)

type DB interface {
	IPLookup(ctx context.Context, ip string) (*Location, error)
}

type City struct {
	// Human readable name in English.
	Name string
}

type Country struct {
	// A two-character ISO 3166-1 country code for the country associated with the location,
	// please refer to https://en.wikipedia.org/wiki/ISO_3166-1.
	IsoCode string

	// Human readable name in English.
	Name string

	IsInEuropeanUnion bool
}

type Subdivision struct {
	// A string of up to three characters containing the region-portion of the ISO 3166-2 code
	// for the first level region associated with the IP address.
	//
	// Some countries have two levels of subdivisions, in which case this is the least specific.
	// For example, in the United Kingdom this will be a country like "England", not a county like "Devon".
	IsoCode string

	// Human readable name in English.
	Name string
}

type Location struct {
	City    City
	Country Country

	// Some countries have one or two levels of subdivisions, see Subdivision
	// [0]: First level
	// [1]: Second level.
	Subdivisions []Subdivision
}
