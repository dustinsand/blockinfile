blockinfile
===========

Command line utility (CLI) that will insert/update/remove a block of multi-line text surrounded by customizable marker
lines.

# Inspiration

If this looks similar to
Ansible's [ansible.builtin.blockinfile](https://docs.ansible.com/ansible/latest/collections/ansible/builtin/blockinfile_module.html),
it should. This CLI is based on it and supports a subset of the Ansible blockinfile flags. This CLI was created for
scenarios
when you wanted Ansible's blockinfile function, but did not want to be dependent on installing Ansible.

# Installation

```
go get github.com/dustinsand/blockinfile
```

# CLI arguments

| Argument | Comments                                        |
|----------|-------------------------------------------------|
| config   | File with blockinfile configuration parameters. |

# Configuration File Parameters

| Parameter       | Choices                           | Comments                                                                                                                                                                                                                        |
|-----------------|-----------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| backup          | true/false Default: false         | Create a backup file including the timestamp information so you can get the original file back if you somehow clobbered it incorrectly.                                                                                         |
| block           | text                              | The text to insert inside the marker lines.                                                                                                                                                                                     |
| group           | text                              | Name of the group that should own the file.                                                                                                                                                                                     |
| indent          | Default: 0                        | The number of spaces to indent the block. Indent must be >= 0.                                                                                                                                                                  |
| insertafter     | text                              | If specified and no begin/ending marker lines are found, the block will be inserted after the last match of specified text. If specified regular expression has no matches, EOF will be used instead.                           |
| insertbefore    | text                              | If specified and no begin/ending marker lines are found, the block will be inserted before the last match of specified text. If specified regular expression has no matches, the block will be inserted at the end of the file. |
| marker          | Default: "# {mark} MANAGED BLOCK" | The marker line template. {mark} will be replaced with the values in marker_begin (default="BEGIN") and marker_end (default="END").                                                                                             |
| markerbegin     | Default: "BEGIN"                  | This will be inserted at {mark} in the opening block marker.                                                                                                                                                                    |
| markerend       | Default: "END"                    | This will be inserted at {mark} in the closing block marker.                                                                                                                                                                    |
| mode            | text                              | The permissions the resulting file should have. For example, '0644' or '0755'.                                                                                                                                                  |
| owner           | text                              | Name of the user that should own the file.                                                                                                                                                                                      |
| path (required) | text                              | The file to modify. If the file does not exist, it will be created.                                                                                                                                                             |
| state           | true/false Default: true          | Whether the block should be there or not.                                                                                                                                                                                       |

# Examples

## Example 1 - Replace block with new text.

/tmp/example1.txt

```text
line 1
line 2
# BEGIN MANAGED BLOCK
old block 1
old block 2
# END MANAGED BLOCK
line 3
line 4
```

/tmp/blockinfile1.yml

```yaml
path: /tmp/example1.txt
block: |-
  new block 1
  new block 2
indent: 2
```

```blockinfile --config /tmp/blockinfile1.yml```

Would update /tmp/example1.txt with

```text
line 1
line 2
  # BEGIN MANAGED BLOCK
  new block 1
  new block 2
  # END MANAGED BLOCK
line 3
line 4
```

## Example 2 - Add block to file.

/tmp/example2.txt

```text
line 1
line 2
line 3
line 4
```

/tmp/blockinfile2.yml

```yaml
path: /tmp/example2.txt
block: |-
  new block 1
  new block 2
indent: 2
insertbefore: "line 1"
```

```blockinfile --config /tmp/blockinfile2.yml```

Would update /tmp/example2.txt with

```text
  # BEGIN MANAGED BLOCK
  new block 1
  new block 2
  # END MANAGED BLOCK
line 1
line 2
line 3
line 4
```