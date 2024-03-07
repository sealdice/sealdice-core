package migrate

import (
	"errors"
	"fmt"
	"os"

	"sealdice-core/utils/crypto"
)

func V144RemoveOldHelpdoc() error {
	const oldSHA256 = "b41430b23d2d59261e496905f7f716040814a6b6646a369a9c3145cc73da3bf4"
	const newSHA256 = "23bb5d258ad33c32183026e8067b9824ee5310204b2c306da623628ce57848b2"
	const oldName = "data/helpdoc/COC/蜜瓜包-怪物之锤查询.json"
	const newName = "data/helpdoc/COC/怪物之锤查询.json"

	_, err := os.Stat(oldName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Get file info for %s failed: %w", oldName, err)
	}

	_, err = os.Stat(newName)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("New helpdoc %s not found. Skip removing old helpdoc %s\n", newName, oldName)
		return nil
	}
	if err != nil {
		return fmt.Errorf("Get file info for %s failed: %w", newName, err)
	}

	if crypto.Sha256Checksum(oldName) != oldSHA256 {
		fmt.Printf("Old helpdoc %s checksum mismatch. You may have edited this file?\n", oldName)
		return nil
	}

	if crypto.Sha256Checksum(newName) != newSHA256 {
		fmt.Printf("New helpdoc %s checksum mismatch. Skip removing old helpdoc %s\n", newName, oldName)
		return nil
	}

	fmt.Printf("Removing old helpdoc %s\n", oldName)
	os.Remove(oldName)
	return nil
}
