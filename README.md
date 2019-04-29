# 0x81

A tool to compare the result set difference of the same query in TiDB/MySQL

![DEMO](./media/demo.png)

## Shortcut

- `TAB` switch focus in different panel
- `ESC` focus SQL input panel
- `UP`  focus on the previous history item
- `DOWN` focus on the next history item
- `ENTER` select current selected history item

## Features

- Record the histories
- Highlight the diff in result sets
- Support `!` directive, which enable the template support for query, e.g:

    `! {{$count=count 10}} insert into t values {{range $index := $count}}({{int 10 100}}){{if head $index $count}},{{end}}{{end}}`
    
- Template functions

    - `count n` returns a slice contains `n` elements
    - `first index slice` returns whether `index` is the first element of the `slice`
    - `last index slice`returns whether `index` is the last element of the `slice`
    - `head index slice`returns whether `index` is the head element of the `slice`
    - `tail index slice`returns whether `index` is the tail element of the `slice`
    - `int min range`returns an int which in `[min, min+range)`
    - `char length`returns a rand string which length is `length`
    - `varchar length`returns a rand string which length in `[length/2, length)`
    
## Examples

1. `create table ttt(a bigint(20) not null auto_increment primary key, b bigint(20), c varchar(10))`
2. `! {{$count:=count 20}} insert into ttt values {{range $index := $count}} (NULL, '{{int 10 100}}', '{{varchar 10}}'){{if head $index $count}},{{end}}{{end}}`