# Ask questions against a website

```bash
WebpageScraper: {"name":"WebPage","parameters{""","url":"https://eldora.com"}}
User: What is the current area open in https://eldora.com?
Mar 19 13:52:35.090 DBG agent.starting agent_name=WebpageScraper max_messages=10 initial_messages_count=3 tools_count=1
Mar 19 13:52:35.090 DBG agent.requesting agent_name=WebpageScraper model=llama3.2 tools_count=1
Mar 19 13:52:35.974 DBG agent.received agent_name=WebpageScraper has_content=false tool_calls_count=1
Mar 19 13:52:35.974 DBG agent.tool_call agent_name=WebpageScraper tool_name=WebPage tool_call_id=call_hckti6o7
Mar 19 13:52:35.974 DBG agent.executing_function agent_name=WebpageScraper tool_name=WebPage
Mar 19 13:52:35.974 DBG tool.call name=WebPage params=map[url:https://eldora.com]
Mar 19 13:52:35.974 DBG tool.call name=WebPage key=url value=https://eldora.com field="<invalid reflect.Value>"
Mar 19 13:52:35.974 DBG tool.call name=WebPage jsonTag=url key=url
Mar 19 13:52:35.974 DBG tool.call name=WebPage instance=&{Url:https://eldora.com}
Mar 19 13:52:42.049 DBG agent.function_result agent_name=WebpageScraper tool_name=WebPage result_type=tools.WebPageResponse
Mar 19 13:52:42.050 DBG agent.requesting agent_name=WebpageScraper model=llama3.2 tools_count=1
Mar 19 13:52:48.903 DBG agent.received agent_name=WebpageScraper has_content=true tool_calls_count=0
Mar 19 13:52:48.903 DBG agent.completed agent_name=WebpageScraper reason=no_tool_calls
Mar 19 13:52:48.903 DBG agent.run_completed agent_name=WebpageScraper final_messages_count=6
WebpageScraper: 
WebPage({"url"="https=//eldora.com"})
WebpageScraper: The current area open in https://eldora.com is 680 acres of alpine terrain with 3 inches of new snow overnight and up to 3 additional inches forecasted today, offering ideal conditions for skiing and riding. There are only five weeks left in the season, so it's a great day to come up and enjoy the mountain while it's less busy.
```