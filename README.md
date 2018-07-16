# gomngr

Part of [gocc](https://github.com/ad/gocc), is responsible for creating measurements and their lifetime

Only for tasks with type "measurement".


# TODO
1. Block task
2. Find the correct number of zonds with the same destination parameter as in main task
3. Create a subtask for each zond (+ set uuid of the main task)
4. Send posts to pubsub with task metadata
5. Wait for a while (timeout/deadline from the main task)
6. Delete / Hide / Mark Unfinished Jobs
7. Make a calculation with data from the completed tasks
8. Write the result to the main task