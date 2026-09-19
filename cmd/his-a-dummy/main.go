package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// patient mirrors the Hospital A response contract from the assignment PDF.
type patient struct {
	PatientHN    string `json:"patient_hn"`
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

// patients covers the shapes the middleware has to cope with: Thai and English
// names, middle names present and absent, and patients carrying only a national
// ID, only a passport, or both.
var patients = []patient{
	{PatientHN: "A0001", FirstNameTH: "เดโม", LastNameTH: "ผู้ป่วย", FirstNameEN: "Demo", LastNameEN: "Patient", DateOfBirth: "1990-01-05", NationalID: "1234567890123", PassportID: "P1234567", PhoneNumber: "0812345678", Email: "demo@example.com", Gender: "M"},
	{PatientHN: "A0002", FirstNameTH: "สมชาย", LastNameTH: "ใจดี", FirstNameEN: "Somchai", LastNameEN: "Jaidee", DateOfBirth: "1985-03-12", NationalID: "1100200300400", PhoneNumber: "0811111111", Email: "somchai@example.com", Gender: "M"},
	{PatientHN: "A0003", FirstNameTH: "สมหญิง", MiddleNameTH: "แก้ว", LastNameTH: "รักเรียน", FirstNameEN: "Somying", MiddleNameEN: "Kaew", LastNameEN: "Rakrian", DateOfBirth: "1992-07-24", NationalID: "1100200300401", PhoneNumber: "0822222222", Email: "somying@example.com", Gender: "F"},
	{PatientHN: "A0004", FirstNameTH: "อนุชา", LastNameTH: "พงษ์ไพบูลย์", FirstNameEN: "Anucha", LastNameEN: "Pongpaiboon", DateOfBirth: "1978-11-02", NationalID: "1100200300402", PassportID: "AA1234567", PhoneNumber: "0833333333", Email: "anucha@example.com", Gender: "M"},
	{PatientHN: "A0005", FirstNameTH: "ปิยะดา", LastNameTH: "ศรีสุข", FirstNameEN: "Piyada", LastNameEN: "Srisuk", DateOfBirth: "2001-02-28", NationalID: "1100200300403", PhoneNumber: "0844444444", Email: "piyada@example.com", Gender: "F"},
	{PatientHN: "A0006", FirstNameEN: "John", MiddleNameEN: "Edward", LastNameEN: "Smith", DateOfBirth: "1969-06-15", PassportID: "GB9876543", PhoneNumber: "0855555555", Email: "john.smith@example.com", Gender: "M"},
	{PatientHN: "A0007", FirstNameEN: "Maria", LastNameEN: "Garcia", DateOfBirth: "1995-09-09", PassportID: "ES5551234", PhoneNumber: "0866666666", Email: "maria.garcia@example.com", Gender: "F"},
	{PatientHN: "A0008", FirstNameTH: "ธนกร", LastNameTH: "วงศ์วาน", FirstNameEN: "Thanakorn", LastNameEN: "Wongwan", DateOfBirth: "1988-12-31", NationalID: "1100200300404", PhoneNumber: "0877777777", Email: "thanakorn@example.com", Gender: "M"},
	{PatientHN: "A0009", FirstNameTH: "กมลชนก", LastNameTH: "ทองดี", FirstNameEN: "Kamonchanok", LastNameEN: "Thongdee", DateOfBirth: "1999-04-18", NationalID: "1100200300405", PassportID: "P7654321", PhoneNumber: "0888888888", Email: "kamon@example.com", Gender: "F"},
	{PatientHN: "A0010", FirstNameTH: "วีระ", LastNameTH: "สุขสันต์", FirstNameEN: "Weera", LastNameEN: "Suksan", DateOfBirth: "1960-08-08", NationalID: "1100200300406", PhoneNumber: "0899999999", Email: "weera@example.com", Gender: "M"},
	{PatientHN: "A0011", FirstNameEN: "Yuki", LastNameEN: "Tanaka", DateOfBirth: "1993-05-05", PassportID: "JP1122334", PhoneNumber: "0900000001", Email: "yuki.tanaka@example.com", Gender: "F"},
	{PatientHN: "A0012", FirstNameTH: "ณัฐพล", MiddleNameTH: "ชัย", LastNameTH: "อินทร์แก้ว", FirstNameEN: "Nattapon", MiddleNameEN: "Chai", LastNameEN: "Inkaew", DateOfBirth: "1983-10-21", NationalID: "1100200300407", PhoneNumber: "0900000002", Email: "nattapon@example.com", Gender: "M"},
}

// byID lets either identifier reach the same patient, which is what the PDF means
// by "id can be either national_id or passport_id".
func byID() map[string]patient {
	index := make(map[string]patient, len(patients)*2)
	for _, p := range patients {
		if p.NationalID != "" {
			index[p.NationalID] = p
		}
		if p.PassportID != "" {
			index[p.PassportID] = p
		}
	}
	return index
}

func main() {
	log.Printf("Dummy Hospital A HIS listening on 127.0.0.1:8081 with %d patients", len(patients))
	log.Fatal(http.ListenAndServe("127.0.0.1:8081", newHandler()))
}

func newHandler() http.Handler {
	index := byID()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /patient/search/{id}", func(w http.ResponseWriter, r *http.Request) {
		found, ok := index[r.PathValue("id")]
		if !ok {
			http.Error(w, `{"error":"patient not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(found)
	})
	return mux
}
