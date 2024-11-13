package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Response struct {
	Reserves []interface{} `json:"reserves"`
	Total    int           `json:"total"`
}

func fetchAllReserves(epgStationURL string) ([]byte, error) {
	// トータルの件数を取得する
	fetchTotalURL := fmt.Sprintf("%s/api/reserves?isHalfWidth=true", epgStationURL)
	fmt.Printf("トータル件数を取得 : %s\n", fetchTotalURL)
	resp, err := http.Get(fetchTotalURL)
	if err != nil {
		return nil, fmt.Errorf("EPGStationからの初回取得に失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("初回取得でエラー発生: ステータスコード %d", resp.StatusCode)
	}

	// レスポンスボディを読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("初回取得のレスポンス読み取りに失敗: %v", err)
	}

	// レスポンスのJSONをデコード
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("初回取得のJSONデコードに失敗: %v", err)
	}

	// 取得件数とtotalが一致する場合は1回目の結果を返す
	if len(response.Reserves) == response.Total {
		return body, nil
	}

	// limitパラメータを追加した2回目のGETリクエスト
	fetchAllReservesURL := fmt.Sprintf("%s/api/reserves?isHalfWidth=true&limit=%d", epgStationURL, response.Total)
	fmt.Printf("全件を取得 : %s\n", fetchAllReservesURL)
	resp2, err := http.Get(fetchAllReservesURL)
	if err != nil {
		return nil, fmt.Errorf("EPGStationからの全件取得に失敗: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("全件取得でエラー発生: ステータスコード %d", resp2.StatusCode)
	}

	// 2回目のレスポンスを読み取り
	body, err = io.ReadAll(resp2.Body)
	if err != nil {
		return nil, fmt.Errorf("全件取得のJSONデコードに失敗: %v", err)
	}

	return body, nil
}

func updateICal(url string, body []byte) error {
	updateICalURL := fmt.Sprintf("%s/update", url)
	resp, err := http.Post(updateICalURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("更新リクエストに失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("更新でエラー発生: ステータスコード %d\n", resp.StatusCode)
		// エラーレスポンスの読み取りと整形出力
		errorBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("エラーレスポンスの読み取りに失敗: %v", err)
		}

		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, errorBody, "", "    "); err != nil {
			fmt.Printf("JSONの整形に失敗: %v\n", err)
			fmt.Println(string(errorBody)) // 整形に失敗した場合は生のJSONを出力
		} else {
			fmt.Println(prettyJSON.String())
		}
		return fmt.Errorf("更新に失敗しました")
	}

	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: epgs-reserves-ical-updater <epgs_url> <ical_update_url>")
		os.Exit(1)
	}

	epgStationURL := os.Args[1]
	icalUpdateURL := os.Args[2]

	// 予約情報を取得
	body, err := fetchAllReserves(epgStationURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// iCalを更新
	if err := updateICal(icalUpdateURL, body); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("更新に成功しました。")
}
