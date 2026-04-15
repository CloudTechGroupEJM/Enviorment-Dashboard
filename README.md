# Assignment 2

## Run and start the application
### Requirements
- Docker
  - Install Docker from [here](https://docs.docker.com/get-docker/)
- Firestore database
  - Create a Firestore database in Google Cloud Platform (GCP) and add the necessary credentials to access it. You can follow the instructions [here](https://cloud.google.com/firestore/docs/quickstart-servers) to set up your Firestore database and obtain the credentials.
- OpenAQ API
  - The application uses the OpenAQ API to fetch air quality data. You can find more information about the OpenAQ API and how to use it [here](https://docs.openaq.org/).

### Steps
1. Clone the repository:
    ```bash
    git clone git@git.gvk.idi.ntnu.no:course/prog2005/prog2005-2026-workspace/msmadhun/Assignment-2.git
    ```
   or
    ```bash
    git clone https://git.gvk.idi.ntnu.no/course/prog2005/prog2005-2026-workspace/msmadhun/Assignment-2.git
    ```
2. Navigate to the project directory.
    ```bash
    cd Assignment-2
    ```
3. Add your Firestore credentials and you OpenAQ API key to the project. 
You can do this by creating a `.env` file in the root directory of the project and adding the following line, 
replacing `path/to/your/credentials.json` with the actual path to your Firestore credentials file and `your_openaq_api_key` with your actual OpenAQ API key:
    ```bash
    GOOGLE_APPLICATION_CREDENTIALS=path/to/your/credentials.json
    OPEN_AQ_API_KEY=your_openaq_api_key
    ```
4. Run the startup script to build and start the Docker container.
    ```bash
    ./startup.sh
    ```
5. The application will be running on `http://localhost:8080`. You can access it through your web browser or use tools like Postman to test the API endpoints.

# Transient dependency
google.golang.org/api v0.272.0
google.golang.org/grpc v1.79.3

