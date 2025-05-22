TheInfiniagen
=

The backup to save the day in case ThePrimeagen loses his voice.

The setup
-

### The eyes

Download the HTML page and extract the article text from the page.

### The brain

Parse the article and add commentary based on the beliefs, values and opinions of ThePrimeagen as expressed in the interview with Lex Friedman.

### The mouth

Translate the article with commentary to speech.

Running it
-

Get your own Google API Key in the Google AI Studio. There's a generous free tier on the 2.5 flash model to play with.

```
export GOOGLE_API_KEY=XXX
go run main.go https://craftofcoding.wordpress.com/2014/04/16/dijkstra-on-ada/
```
