package store

import (
	"cloud.google.com/go/firestore"
	"context"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"log"
)

// Firebase context and client used by Firestore functions throughout the program.
var firebaseContext context.Context


//Returns Firebase context and initializes if not already done.
func GetFirebaseContext() context.Context {
	if firebaseContext == nil {
		firebaseContext = context.Background()
	}
	return firebaseContext
}


//Initializes Firebase client and returns reference.
//Returns error if problems during initialization occur.
func GetFirebaseClient() (*firestore.Client, error) {
	// Firebase initialisation
	firebaseContext = GetFirebaseContext()

	// We use a service account, load credentials file that you downloaded from your project's settings menu.
	// It should reside in your project directory.
	// Make sure this file is git-ignored, since it is the access token to the database.
	credentialsOption := option.WithCredentialsFile("../credentials/db.json")
	firebaseApp, firebaseAppInitError := firebase.NewApp(firebaseContext, nil, credentialsOption)
	if firebaseAppInitError != nil {
		log.Println(firebaseAppInitError)
		return nil, firebaseAppInitError
	}

	// Instantiate client
	client, firestoreClientInitError := firebaseApp.Firestore(firebaseContext)

	// Alternative setup, directly through Firestore (without initial reference to Firebase); but requires Project ID; useful if multiple projects are used
	// client, err := firestore.NewClient(firebaseContext, projectID)

	// Check whether there is an error when connecting to Firestore
	if firestoreClientInitError != nil {
		log.Println(firestoreClientInitError)
		return client, firestoreClientInitError
	}

	return client, nil
}



