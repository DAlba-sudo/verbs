# Verbs: General Components for Verb

[Verb](https://github.com/DAlba-sudo/verb) is an HTMX-centric website builder. *Verbs* are general components that you can use in 
your own Verb projects. 

| Name | Description |
| - | - |
| [TicketQ](./TicketQueue.go) | This is a "Bridge" that you can add to your verb "Component". It controls the amount of concurrent operations for a specific route. I've used this to limit expensive concurrent operations. The ticket queue is managed by a go channel! It modifies the "Htmx" payload read by the Component template to instruct the HTMX element to come back later via [Load Polling](https://htmx.org/docs/#load_polling). |
