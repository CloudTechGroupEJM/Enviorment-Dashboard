package structs

import (
	"cloud.google.com/go/firestore"
)


// Handler holds shared dependencies
type FSClient struct {
	Client *firestore.Client
}
