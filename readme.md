## PRICE PARSER 
Abandoned pet-project to get acquainted with the Go programming language.
Some good practices are not supported here.
The project is workable and ready to be used.

## How to use
You need to prepare the `pricewatcher.db` before use the app. 
First, populate the `shops` table:
- `id` - unique identifier;
- `shops.title` - shop name; e.g., aliexpress, ozon, etc.
- `title_x_path`, `picture_x_path`, `price_x_path` - [x path](https://en.wikipedia.org/wiki/XPath) of the html element that stores the data to be parsed;
for example it could be taken from the google chrome html inspector.

Second, populate the `items` table by data you want to parse:
- `id` - uniqure identifier;
- `items.title` - name of the item you want to parse; e.g., ps5, iphoneN, etc.;
- `link` - url to the item page in a web store;
- `shop_id` - `id` of the record from the `shops` table;

Third. `go run main.go`
Results can be found in the `prices` table.
btw there should be the `log/` directory in the `main.go` folder:
- `%YYYY-mm-dd_hh:mm%.log` - is the output from the main.go execution;
- `%timestamp%/%shop name%/%item title%/` - directory contains received screenshot and html files of the page to be parsed; 

## How to deploy
There is a dockerfile I've checked only once :]

## How to schedule
main function already has (but commented) usage of the [cron lib for go](https://github.com/robfig/cron). You can add it to depends and uncomment section from the `main()` function.

I used crontab on ubuntu server:
`crontab -e`

add this line at the end:
`0 */3 * * * cd root/workspace/priceparser && bash launch.sh`

`launch.sh` with the following content:
```
#!/bin/bash
cd /root/workspace/priceparser/
go run main.go >> "log/$(date +%Y-%m-%d_%H:%M).log"
```

## Dependencies
I almost forgot. [chromedp](https://github.com/chromedp) requires chrome to be installed on your system :] I believe you just need to install [headless-chrome](https://developers.google.com/web/updates/2017/04/headless-chrome) for linux server. 
I think I used this [github-gist](https://gist.github.com/ipepe/94389528e2263486e53645fa0e65578b) to install this on my ubuntu server.

## TODO
- [x] - add table to store parsed values;
- [x] - add connection from go to sqlite db;
- [x] - read/write data from/to sqlite db;
- [x] - receive shop lists from db;
- [x] - receive shop items from db;
- [x] - fill the database with source data that will need to be parsed;
- [X] - select and prepare data to parse;
- [x] - parse only 1 element (price w/o title, desc, etc) from the source;  
- [x] - can't receive data from aliexpress;
- [x] - prepare structure to store into db parsed data;
- [x] - remove letters from parsed price;
- [x] - do not forget to store original (not filtered) price into db;
- [x] - add parsing data into db;
- [x] - store screeshot and logs paths into db;
- [x] - use proxy;
- [] - deploy via docker;
- [] - use this cron [library](https://github.com/robfig/cron)
- [] - can use datadog to check logs;
- [] - add concurrent execution (no need to parse data sequentially);
- [] - increase timeout between requests to the same domains;
- [] - need to switch emulated-clients periodically;
