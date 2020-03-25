package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	lim "github.com/korovkin/limiter"
	"log"
	"math"
	"net/http"
	"time"
)

var BadCombinations [][]bool
var Combination []bool

var Port = "55001"
var Version = "0.0.1"

var processCount int

type Numbers []Number

func (a Numbers) Len() int           { return len(a) }
func (a Numbers) Less(i, j int) bool { return a[i].Value > a[j].Value }
func (a Numbers) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type Number struct {
	Value    int    `json:"value"`
	NumberId string `json:"id"`
}

type OutNumber struct {
	Value    int    `json:"value"`
	NumberId string `json:"id"`
	Selected bool   `json:"bool"`
}

type IncomingHTTPRequest struct {
	Amount  int      `json:"amount"`
	Numbers []Number `json:"numbers"`
}

type OutgoingHTTPResponse struct {
	Amount  int         `json:"amount"`
	Numbers []OutNumber `json:"numbers"`
}

var exitByCondition int

//var dirs chan bool

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", TryToPickNumbers)

	//dirs = make(chan bool)
	//
	//go func() {
	//	for {
	//		select {
	//		case dir := <-dirs:
	//			if dir {
	//				processCount += 1
	//			} else {
	//				processCount -= 1
	//			}
	//		}
	//	}
	//}()

	fmt.Printf("Starting Server to HANDLE programmatic.tech back end\nPort : " + Port + "\nAPI revision " + Version + "\n\n")
	if err := http.ListenAndServe(":"+Port, r); err != nil {
		log.Fatal(err, "ERROR")
	}
}

func TryToPickNumbers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var incomingData IncomingHTTPRequest
		err := json.NewDecoder(r.Body).Decode(&incomingData)
		if err != nil {
			ResponseBadRequest(w, err, "")
			return
		}
		//sort.Sort(Numbers(incomingData.Numbers))

		PrintIncomingData(incomingData)

		FoundCombination := getAnswer(incomingData)
		var outgoingData OutgoingHTTPResponse
		for i := 0; i < len(incomingData.Numbers); i++ {
			outgoingData.Numbers = append(outgoingData.Numbers, OutNumber{
				Value:    incomingData.Numbers[i].Value,
				NumberId: incomingData.Numbers[i].NumberId,
				Selected: FoundCombination[i],
			})
			if FoundCombination[i] {
				outgoingData.Amount += incomingData.Numbers[i].Value
			}
		}

		response, _ := json.Marshal(outgoingData)
		ResponseOK(w, response)
		return
	default:
		ResponseBadRequest(w, nil, "{\"message\":\"method not found\"}")
		return
	}
}

func bin2dec(ar []bool) uint64 {
	var res uint64
	for i := len(ar) - 1; i >= 0; i-- {
		if ar[i] == true {
			res += uint64(math.Pow(float64(2), float64(len(ar)-1-i)))
		}
	}
	return res
}

func Check(t1 time.Time, comb []bool, id IncomingHTTPRequest) (uint64, bool) {

	ifBad := make([]bool, len(id.Numbers))

	var am int
	for pos := 0; pos < len(comb); pos++ {
		if comb[pos] {
			am += id.Numbers[pos].Value
			ifBad[pos] = true
		}

		if am > id.Amount {
			return bin2dec(ifBad), false
		}
	}
	if am == id.Amount {
		fmt.Println(bin2str(comb), " Tooks : ", time.Now().Sub(t1))
		return 0, true
	}
	return 0, false
}

func bin2str(ar []bool) string {
	var s string
	for _, el := range ar {
		if el {
			s += "1"
		} else {
			s += "0"
		}
	}
	return s
}

func getAnswer(id IncomingHTTPRequest) []bool {

	t1 := time.Now()
	count := uint64(PossiblePlaces(len(id.Numbers)))

	limit := lim.NewConcurrencyLimiter(8)

	var am int
	var fromBottom int
	for k := len(id.Numbers) - 1; k >= 0; k-- {
		am += id.Numbers[k].Value
		fromBottom += 1
		if am >= id.Amount {
			break
		}
	}

	fmt.Println("We need to truncate ", fromBottom, "elements", PossiblePlaces(fromBottom), "combinations")
	minCount := uint64(PossiblePlaces(fromBottom))
	fmt.Println("We need to check ", count-minCount, "combinations")

	minCount = 0

	//for i:=count; i>=0; i-- {
	//	places = append(places, strconv.FormatInt(int64(i), 2))
	//}

	for {
		Combination = MakeCombination(count, len(id.Numbers))
		n_count, flag := Check(t1, Combination, id)
		if flag {
			return Combination
		}
		if n_count != 0 {
			count = n_count - 1
		} else {
			count -= 1
		}
		if count <= 0 {
			break
		}
	}

	limit.Wait()

	fmt.Println("Final Tooks : ", time.Now().Sub(t1))
	fmt.Println("exitByCondition", exitByCondition)
	return make([]bool, len(id.Numbers))
}

func MakeCombination(numb uint64, length int) []bool {
	bin := make([]bool, length)
	for {
		if numb%2 == 1 {
			bin[length-1] = true
			numb = (numb - 1) / 2
			length--
		} else {
			numb = numb / 2
			length--
		}
		if length == 0 {
			break
		}
	}
	return bin
}

func PrintIncomingData(id IncomingHTTPRequest) {
	fmt.Println("Amount : ", id.Amount)
	fmt.Println("Numbers : ")
	for ind, item := range id.Numbers {
		fmt.Println("\t", ind+1, item.Value, item.NumberId)
	}
	fmt.Println("Number of possible perturbation : ", PossiblePlaces(len(id.Numbers)))
}

func PossiblePlaces(count int) (result int) {
	for i := 0; i < count; i++ {
		result += int(math.Pow(2, float64(i)))
	}
	return result
}

func ResponseOK(w http.ResponseWriter, addedRecordString []byte) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	n, _ := fmt.Fprintf(w, string(addedRecordString))
	fmt.Println("Response was sent ", n, " bytes")
	return
}

func ResponseBadRequest(w http.ResponseWriter, err error, message string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	errorString := "{\"error_message\":\"" + err.Error() + "\",\"message\":\"" + message + "\"}"
	http.Error(w, errorString, http.StatusBadRequest)
	return
}

func ResponseNotFound(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	n, _ := fmt.Fprintf(w, "")
	fmt.Println("Response was sent ", n, " bytes")
	return
}
