package structs

// APIKeyRegistration represents a client registration request
// Fields: 
//   - name: Client application name (required)
//   - email: Client contact email (required)
type APIKeyRegistration struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// APIKeyResponse represents the API key issued to a client
// Fields:
//   - key: Generated API key (sk-envdash-{random})
//   - createdAt: Timestamp when key was created (format: "20250301 09:15")
type APIKeyResponse struct {
    Key       string `json:"key"`
    CreatedAt string `json:"createdAt"`
}

// APIKeyStoreModel represents how API keys are stored in Firestore
// Fields:
//   - id: Document ID
//   - key: The actual API key
//   - name: Client application name
//   - email: Client contact email
//   - createdAt: Timestamp in DATE_FORMAT
//   - isActive: Whether the key is currently valid
//   - lastUsed: Timestamp of last API call using this key
type APIKeyStoreModel struct {
    ID        string `firestore:"id,omitempty" json:"id"`
    Key       string `firestore:"key" json:"key"`
    Name      string `firestore:"name" json:"name"`
    Email     string `firestore:"email" json:"email"`
    CreatedAt string `firestore:"createdAt" json:"createdAt"`
    IsActive  bool   `firestore:"isActive" json:"isActive"`
    LastUsed  string `firestore:"lastUsed" json:"lastUsed"`
}