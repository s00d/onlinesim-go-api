# Free numbers

> Public numbers and their SMS — client.Free(). No API key needed.

## List

Countries, the numbers of the selected country and the SMS of the selected number, in one call.

```go
func (f *FreeAPI) List(ctx context.Context, opts FreeListOptions) (FreeList, error)
```

<table>
<thead>
  <tr>
    <th>
      <code>
        FreeListOptions
      </code>
      
       field
    </th>
    
    <th>
      Type
    </th>
    
    <th>
      Zero value
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
        Country
      </code>
    </td>
    
    <td>
      <code>
        int
      </code>
    </td>
    
    <td>
      server: <code>
        44
      </code>
    </td>
    
    <td>
      Country code; an unknown one falls back to the first available.
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
        string
      </code>
    </td>
    
    <td>
      first number
    </td>
    
    <td>
      Number <strong>
        without
      </strong>
      
       the country code, as the keys of <code>
        Numbers
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Page
      </code>
    </td>
    
    <td>
      <code>
        int
      </code>
    </td>
    
    <td>
      <code>
        1
      </code>
    </td>
    
    <td>
      Page of <code>
        Messages
      </code>
      
      .
    </td>
  </tr>
  
  <tr>
    <td>
      <code>
        Count
      </code>
    </td>
    
    <td>
      <code>
        int
      </code>
    </td>
    
    <td>
      server default
    </td>
    
    <td>
      SMS per page, 10–500.
    </td>
  </tr>
</tbody>
</table>

Zero fields are not sent.

`FreeList`:

- `Countries`: `[]FreeCountry{Country, CountryText, CountryOriginal}`
- `Numbers`: `map[string]FreeNumber{Country, CountryOriginal, FullNumber, DataHumans, IsArchive}`, keyed by the short number
- `Messages`: a `Paginator[FreeMessage]` (`{ID, Text, InNumber, MyNumber, Code, CreatedAt, DataHumans}`) plus `Number` and `Country`; `Code` is `-` when no code was found

```go
free := client.Free()
list, err := free.List(ctx, onlinesim.FreeListOptions{Country: 44})
if err != nil {
    return err
}
for short := range list.Numbers {
    page, err := free.List(ctx, onlinesim.FreeListOptions{Country: 44, Number: short, Count: 20})
    if err != nil {
        return err
    }
    for _, sms := range page.Messages.Data {
        fmt.Println(sms.InNumber, sms.Code, sms.Text)
    }
    break
}
```
