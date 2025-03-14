package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/firestore", firestoreHandler)
	http.HandleFunc("/firestore/", firestoreHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, GIG!")
}

func firestoreHandler(w http.ResponseWriter, r *http.Request) {

	// Firestore クライアント作成
	pid := os.Getenv("GOOGLE_CLOUD_PROJECT")
	ctx := r.Context()
	client, err := firestore.NewClient(ctx, pid)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	switch r.Method {
	// 追加処理
	case http.MethodPost:
		u, err := getUserBody(r)
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ref, _, err := client.Collection("users").Add(ctx, u)
		if err != nil {
			log.Fatalf("Failed adding data: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Print("success: id is %v", ref.ID)
		fmt.Fprintf(w, "success: id is %v \n", ref.ID)

	// 取得処理
	case http.MethodGet:
		id := strings.TrimPrefix(r.URL.Path, "/firestore/")
		log.Printf("id=%v", id)
		if id == "/firestore" || id == "" {
			docs, err := client.Collection("users").Documents(ctx).GetAll()
			if err != nil {
				log.Fatal(err)
			}
			if len(docs) == 0 {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			var users []User
			for _, doc := range docs {
				var u User
				if err := doc.DataTo(&u); err != nil {
					log.Fatal(err)
				}
				u.ID = doc.Ref.ID
				log.Print(u)
				users = append(users, u)
			}
			json, err := json.Marshal(users)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(json)
			return
		}
		// (Step 29) 置き換えここから
		doc, err := client.Collection("users").Doc(id).Get(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var u User
		if err := doc.DataTo(&u); err != nil {
			log.Fatal(err)
		}
		u.Id = doc.Ref.ID
		json, err := json.Marshal(u)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(json)
		// (Step 29) 置き換えここまで

	// 更新処理
	case http.MethodPut:
		id := strings.TrimPrefix(r.URL.Path, "/firestore/")
		u, err := getUserBody(r)
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = client.Collection("users").Doc(id).Set(ctx, u)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "success updating")

	// 削除処理
	case http.MethodDelete:
		id := strings.TrimPrefix(r.URL.Path, "/firestore/")
		_, err := client.Collection("users").Doc(id).Delete(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "success deleting")

	// それ以外のHTTPメソッド
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

type User struct {
	ID    string `firestore:"-" json:"id"`
	Email string `firestore:"email" json:"email"`
	Name  string `firestore:"name" json:"name"`
}

func getUserBody(r *http.Request) (u User, err error) {
	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		return u, err
	}

	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		return u, err
	}

	//parse json
	err = json.Unmarshal(body[:length], &u)
	if err != nil {
		return u, err
	}
	log.Print(u)
	return u, nil
}
