# Planer

Planer is a lightweight go tool that allows for searching logs by a time frame or 
certain words. This is accomplished by chunking the log file into bits and doing this
with some concurrency. This also allows for very large logs to be processed quickly.

### Improvements

- provide a UI to ease the input process
- provide a file structure to allow for planed file output
- provide further ways of quickly planing down logs to get what you need