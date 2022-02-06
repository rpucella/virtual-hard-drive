
package storage

import (
	"fmt"
)

// Convert a UID to a path on Cloud Storage.
// E.g.,
//   7b5d41cc-86d6-11eca8a3-0242ac120002
// to
//   7b/5d/41/cc/7b5d41cc-86d6-11eca8a3-0242ac120002
func UIDToPath(uid string) (string, error) {
	if len(uid) != 36 {
		return "", fmt.Errorf("Length of UUID %s <> 36", uid)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", uid[:2], uid[2:4], uid[4:6], uid[6:8], uid), nil
}
