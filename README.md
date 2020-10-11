# Planer

Planer is a lightweight go tool that allows for searching logs by a time frame or 
certain words. This is accomplished by chunking the log file into bits and doing this
with some concurrency. This also allows for very large logs to be processed quickly.

### Improvements

- provide a UI to ease the input process
- provide a file structure to allow for planed file output
- provide further ways of quickly planing down logs to get what you need

### Object Proposal

Consider creating a quick config file that will allow a user to define their log entries to further power Planer. 
Things that could be outlined are timestamp structure, overall line structure, delimiters, headers, and etc. In 
some perfect word, we could have a user copy-paste a line from their log and Planer would be intelligent enough
to parse out the structure. I think we start with a very railed input from the user and see if we can abstract this 
in any way in the future for the intelligent portion.