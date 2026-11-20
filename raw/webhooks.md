# Webhooks

> Receive SMS as HTTP callbacks and parse them with ParseWebhook.

Set a URL with [`client.User().SetWebhookURL`](/user#setwebhookurl). For each SMS the server POSTs a JSON body to it: `webhook_type` `receiving_sms` for numbers, `rent_sms` for rents. Delivery attempts are listed by [`client.User().WebhookLogs`](/user#webhooklogs).

## ParseWebhook

Parse an inbound webhook body.

```go
func ParseWebhook(data []byte) (WebhookPayload, error)
```

Lenient: numeric fields may arrive as strings, missing or malformed fields become zero values (`""`, `0`), unknown fields are ignored. It returns an error only when the input is not a JSON object.

```go
package main

import (
    "io"
    "log"
    "net/http"

    onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func main() {
    http.HandleFunc("/onlinesim-hook", func(w http.ResponseWriter, r *http.Request) {
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        hook, err := onlinesim.ParseWebhook(body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        if hook.WebhookType == onlinesim.WebhookReceivingSMS {
            log.Println(hook.OperationID, hook.Code)
        }
        _, _ = io.WriteString(w, "ok")
    })
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Payload fields

<table>
<thead>
  <tr>
    <th>
      Field
    </th>
    
    <th>
      JSON
    </th>
    
    <th>
      Type
    </th>
    
    <th>
      Description
    </th>
  </tr>
</thead>

<tbody>
  <tr>
    <td>
      <code>
        WebhookType
      </code>
    </td>
    
    <td>
      <code>
        webhook_type
      </code>
    </td>
    
    <td>
      <code>
        WebhookType
      </code>
    </td>
    
    <td>
      <code>
        WebhookReceivingSMS
      </code>
      
      , <code>
        WebhookRentSMS
      </code>
      
      , or <code>
        ""
      </code>
      
       for anything else.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        OperationID
      </code>
    </td>
    
    <td>
      <code>
        operation_id
      </code>
    </td>
    
    <td>
      <code>
        int64
      </code>
    </td>
    
    <td>
      <code>
        tzid
      </code>
      
       of the operation or rent.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        UserID
      </code>
    </td>
    
    <td>
      <code>
        user_id
      </code>
    </td>
    
    <td>
      <code>
        int64
      </code>
    </td>
    
    <td>
      Account id.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        CountryCode
      </code>
    </td>
    
    <td>
      <code>
        country_code
      </code>
    </td>
    
    <td>
      <code>
        int
      </code>
    </td>
    
    <td>
      Country code of the number.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Number
      </code>
    </td>
    
    <td>
      <code>
        number
      </code>
    </td>
    
    <td>
      <code>
        string
      </code>
    </td>
    
    <td>
      Numbers: <code>
        +
      </code>
      
       and the full number (<code>
        +79001234567
      </code>
      
      ), or <code>
        hidden
      </code>
      
      . Rent: the full number in digits without <code>
        +
      </code>
      
       (<code>
        79001234567
      </code>
      
      ).
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Sender
      </code>
    </td>
    
    <td>
      <code>
        sender
      </code>
    </td>
    
    <td>
      <code>
        string
      </code>
    </td>
    
    <td>
      Sender of the SMS, e.g. <code>
        VK
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Message
      </code>
    </td>
    
    <td>
      <code>
        message
      </code>
    </td>
    
    <td>
      <code>
        string
      </code>
    </td>
    
    <td>
      Full SMS text.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Code
      </code>
    </td>
    
    <td>
      <code>
        code
      </code>
    </td>
    
    <td>
      <code>
        string
      </code>
    </td>
    
    <td>
      Code extracted from the SMS; <code>
        ""
      </code>
      
       when none was found.
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        TimeStart
      </code>
    </td>
    
    <td>
      <code>
        time_start
      </code>
    </td>
    
    <td>
      <code>
        string
      </code>
    </td>
    
    <td>
      Numbers: when the operation was ordered. Rent: when the SMS arrived. <code>
        YYYY-MM-DD HH:MM:SS
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        TimeLeft
      </code>
    </td>
    
    <td>
      <code>
        time_left
      </code>
    </td>
    
    <td>
      <code>
        int64
      </code>
    </td>
    
    <td>
      Time left: seconds for numbers, minutes for rent.
    </td>
  </tr>
</tbody>
</table>
