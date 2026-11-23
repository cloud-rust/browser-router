package main

// processAlive is not needed on Windows, where the lock file itself tells whether Vivaldi runs.
func processAlive(int) bool { return true }
