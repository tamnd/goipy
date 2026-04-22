import readline

# --- backend ---
print(readline.backend)

# --- history management ---
readline.clear_history()
print(readline.get_current_history_length())

readline.add_history('first line')
readline.add_history('second line')
readline.add_history('third line')
print(readline.get_current_history_length())

# get_history_item is 1-based
print(readline.get_history_item(1))
print(readline.get_history_item(2))
print(readline.get_history_item(3))

# replace and remove
readline.replace_history_item(1, 'replaced line')
print(readline.get_history_item(2))

readline.remove_history_item(0)
print(readline.get_current_history_length())
print(readline.get_history_item(1))

# --- history length setting ---
readline.set_history_length(100)
print(readline.get_history_length())
readline.set_history_length(-1)
print(readline.get_history_length())

# --- line buffer ---
print(repr(readline.get_line_buffer()))
readline.insert_text('hello')
readline.redisplay()

# --- completer ---
print(readline.get_completer() is None)

def my_completer(text, state):
    return None

readline.set_completer(my_completer)
print(readline.get_completer() is my_completer)

readline.set_completer()
print(readline.get_completer() is None)

# --- completer delims ---
delims = readline.get_completer_delims()
print(type(delims).__name__)
print(len(delims) > 0)

readline.set_completer_delims('\t\n')
print(readline.get_completer_delims())

# --- completion index ---
print(readline.get_begidx())
print(readline.get_endidx())
print(readline.get_completion_type())

# --- startup hooks ---
readline.set_startup_hook()
readline.set_pre_input_hook()

def hook():
    pass

readline.set_startup_hook(hook)
readline.set_pre_input_hook(hook)
readline.set_startup_hook()
readline.set_pre_input_hook()

# --- set_auto_history ---
readline.set_auto_history(True)
readline.set_auto_history(False)

# --- completion display matches hook ---
readline.set_completion_display_matches_hook()

# --- parse_and_bind (no-op stub) ---
readline.parse_and_bind('tab: complete')

# --- read_init_file (no-op stub) ---
try:
    readline.read_init_file()
except OSError:
    print('read_init_file OSError')

print('done')
