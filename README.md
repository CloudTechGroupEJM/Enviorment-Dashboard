# Assignment 2

## Run and start the application
### Requirements
- Docker
  - Install Docker from [here](https://docs.docker.com/get-docker/)
- Firestore database
  - Create a Firestore database in Google Cloud Platform (GCP) and add the necessary credentials to access it. You can follow the instructions [here](https://cloud.google.com/firestore/docs/quickstart-servers) to set up your Firestore database and obtain the credentials.

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
3. Add the Firestore credentials to the project. Add the file to a folder called `credentials` in the root of the project. The file should be named `db.json`. The path should be: `./credentials/db.json`
4. Run the startup script to build and start the Docker container.
    ```bash
    ./startup.sh
    ```
5. The application will be running on `http://localhost:8080`. You can access it through your web browser or use tools like Postman to test the API endpoints.