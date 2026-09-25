package bot

import (
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func createItems(amount int) []media.Item {
	items := make(
		[]media.Item,
		amount,
	)

	for i := range items {
		items[i] = media.Item{
			Type: media.Photo,
			Path: "photo.jpg",
		}
	}

	return items
}

func TestSplitTwoItems(t *testing.T) {
	batches := splitMediaBatches(
		createItems(2),
	)

	if len(batches) != 1 {
		t.Fatalf(
			"esperava 1 álbum; recebido %d",
			len(batches),
		)
	}

	if len(batches[0]) != 2 {
		t.Fatalf(
			"esperava 2 mídias; recebido %d",
			len(batches[0]),
		)
	}
}

func TestSplitTenItems(t *testing.T) {
	batches := splitMediaBatches(
		createItems(10),
	)

	if len(batches) != 1 {
		t.Fatalf(
			"esperava 1 álbum; recebido %d",
			len(batches),
		)
	}

	if len(batches[0]) != 10 {
		t.Fatalf(
			"esperava 10 mídias; recebido %d",
			len(batches[0]),
		)
	}
}

func TestSplitElevenItems(t *testing.T) {
	batches := splitMediaBatches(
		createItems(11),
	)

	if len(batches) != 2 {
		t.Fatalf(
			"esperava 2 álbuns; recebido %d",
			len(batches),
		)
	}

	if len(batches[0]) != 9 {
		t.Fatalf(
			"primeiro álbum deveria ter 9 mídias; recebido %d",
			len(batches[0]),
		)
	}

	if len(batches[1]) != 2 {
		t.Fatalf(
			"segundo álbum deveria ter 2 mídias; recebido %d",
			len(batches[1]),
		)
	}
}

func TestSplitNeverCreatesSingleItemBatch(t *testing.T) {
	for amount := 2; amount <= 50; amount++ {
		batches := splitMediaBatches(
			createItems(amount),
		)

		for _, batch := range batches {
			if len(batch) == 1 {
				t.Fatalf(
					"%d mídias produziram álbum unitário",
					amount,
				)
			}

			if len(batch) > maxMediaGroupSize {
				t.Fatalf(
					"álbum excedeu limite: %d",
					len(batch),
				)
			}
		}
	}
}
