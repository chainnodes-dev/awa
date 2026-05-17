# Process Description: Crypto Market Sentiment Tracker

## Abstract
Automatically fetches current statistics (price, 24h volume, price change) for the top 10 most traded cryptocurrency coins, analyzes recent market news and social posts to determine the overall crypto market sentiment as well as individual sentiment for each coin, and compiles a comprehensive dashboard report.

## Blackboard Schema Requirements
The blackboard must hold the following strictly typed variables to track state:
- `target_coins` (list of strings, required): A list of the 10 coin tickers to track (e.g., BTC, ETH, SOL).
- `raw_market_data` (object): The raw prices and stats fetched from the market API or search.
- `news_articles` (list of objects): Recent headlines and summaries gathered for context.
- `overall_market_sentiment` (string): The summary market sentiment (e.g., "Bullish", "Bearish", "Fearful", "Greedy").
- `coin_sentiments` (object): Key-value pairs matching each coin ticker to its specific sentiment rating.
- `final_report` (string): A beautifully compiled Markdown report ready for display or distribution.

## Step-by-Step Workflow Specification

1. **Start State (Fetch Market Statistics):**
   - Name: `fetch_coin_stats`
   - Type: `prompt`
   - Target Agent: `AnalystAgent`
   - Description: Using a crypto API MCP server or Brave Search MCP, fetch the current price, 24h trading volume, and 24h percentage change for the coins defined in `target_coins`. Write the structured JSON data into `raw_market_data` on the blackboard.
   - Transitions:
     - On success (`trigger: success`), move to `fetch_sentiment_news`.
     - On failure (`trigger: error`), transition to `terminal_failure`.

2. **Data Gathering State (Gather Social & News Context):**
   - Name: `fetch_sentiment_news`
   - Type: `prompt`
   - Target Agent: `AnalystAgent`
   - Description: Query the Brave Search MCP or Reddit MCP for recent news articles, blog posts, and community discussions relating to the `target_coins`. Save a cleaned list of headlines and article snippets into `news_articles`.
   - Transitions:
     - On complete (`trigger: done`), move to `evaluate_overall_sentiment`.

3. **Sentiment Analysis State (Overall Market Assessment):**
   - Name: `evaluate_overall_sentiment`
   - Type: `prompt`
   - Target Agent: `SentimentAgent`
   - Description: Read `news_articles` and `raw_market_data`. Synthesize the general market mood, calculate the fear and greed metric, and write a summary paragraph along with the classification label into `overall_market_sentiment`.
   - Transitions:
     - On complete (`trigger: done`), move to `evaluate_individual_sentiments`.

4. **Detailed Analysis State (Coin-Specific Sentiments):**
   - Name: `evaluate_individual_sentiments`
   - Type: `prompt`
   - Target Agent: `SentimentAgent`
   - Description: For each coin in `target_coins`, evaluate its specific price action and mention-sentiment in the news articles. Generate a list of key sentiment scores (on a scale of -10 to +10) and specific bullish/bearish reasons, writing them to `coin_sentiments`.
   - Transitions:
     - On complete (`trigger: done`), move to `generate_market_report`.

5. **Report Compilation State (Generate Markdown Dossier):**
   - Name: `generate_market_report`
   - Type: `prompt`
   - Target Agent: `AnalystAgent`
   - Description: Read all compiled data on the blackboard (`raw_market_data`, `overall_market_sentiment`, `coin_sentiments`). Write a visually stunning Markdown report containing summary tables, sentiment indicator badges, and highlights of top performing coins. Write this Markdown to `final_report`.
   - Transitions:
     - On generation complete (`trigger: done`), transition to `terminal_success`.

6. **Terminal States:**
   - `terminal_success` (Type: `terminal`): The crypto market sentiment report was successfully generated and saved to the blackboard.
   - `terminal_failure` (Type: `terminal`): The workflow failed to fetch stats or analyze news resources.
