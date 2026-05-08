package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// extractStringSet extracts a []string from a types.Set.
func extractStringSet(ctx context.Context, set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var result []string
	set.ElementsAs(ctx, &result, false)
	return result
}

// buildStringSet creates a types.Set from a []string.
func buildStringSet(ctx context.Context, items []string) types.Set {
	if len(items) == 0 {
		return types.SetNull(types.StringType)
	}
	elems := make([]types.String, len(items))
	for i, item := range items {
		elems[i] = types.StringValue(item)
	}
	set, _ := types.SetValueFrom(ctx, types.StringType, elems)
	return set
}

// diffSets computes the difference between old and new string slices.
// Returns (toAdd, toRemove).
func diffSets(oldItems, newItems []string) (toAdd, toRemove []string) {
	oldMap := make(map[string]bool, len(oldItems))
	for _, item := range oldItems {
		oldMap[item] = true
	}

	newMap := make(map[string]bool, len(newItems))
	for _, item := range newItems {
		newMap[item] = true
	}

	for _, item := range newItems {
		if !oldMap[item] {
			toAdd = append(toAdd, item)
		}
	}

	for _, item := range oldItems {
		if !newMap[item] {
			toRemove = append(toRemove, item)
		}
	}

	return toAdd, toRemove
}
