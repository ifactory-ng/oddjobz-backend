package main

//Images makes dealing with thumbnails and full size images easier
type Images struct {
	Thumb string `json:"thumb"`
	Full  string `json:"full"`
}

//lookUp holds reference data liking a providers collection eith the users
//collection
type lookUp struct {
	Provider    string
	ProviderUID string
	UserID      string
}
