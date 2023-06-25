package main

import (
	"cache2go"
	"fmt"
	"time"
)

func main() {
	cache := cache2go.Cache("myCache")

	// This callback will be triggered every time a new item
	// gets added to the cache.
	cache.SetAddedItemCallback(func(entry *cache2go.CacheItem) {
		fmt.Println("Added Callback 1:", entry.Key(), entry.Data(), entry.CreateTime())
	})
	cache.AddAddedItemCallback(func(entry *cache2go.CacheItem) {
		fmt.Println("Added Callback 2:", entry.Key(), entry.Data(), entry.CreateTime())
	})
	// This callback will be triggered every time an item
	// is about to be removed from the cache.
	cache.SetDeleteItemCallback(func(entry *cache2go.CacheItem) {
		fmt.Println("Deleting:", entry.Key(), entry.Data(), entry.CreateTime())
	})

	// Caching a new item will execute the AddedItem callback.
	cache.Add("someKey", "This is a test!", 0)

	// Let's retrieve the item from the cache
	res, err := cache.Value("someKey")
	if err == nil {
		fmt.Println("Found value in cache:", res.Data())
	} else {
		fmt.Println("Error retrieving value from cache:", err)
	}

	// Deleting the item will execute the AboutToDeleteItem callback.
	cache.Delete("someKey")

	cache.RemoveAddedItemCallBack()
	// Caching a new item that expires in 3 seconds
	res = cache.Add("anotherKey", "This is another test", 3*time.Second)

	// This callback will be triggered when the item is about to expire
	res.SetAboutToExpireCallback(func(key interface{}) {
		fmt.Println("About to expire:", key.(string))
	})

	time.Sleep(5 * time.Second)
}
