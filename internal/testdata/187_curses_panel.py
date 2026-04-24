"""Tests for curses.panel module."""
import curses
import curses.panel

# --- Module-level functions exist ---
print(callable(curses.panel.new_panel))        # True
print(callable(curses.panel.top_panel))        # True
print(callable(curses.panel.bottom_panel))     # True
print(callable(curses.panel.update_panels))    # True

# --- panel class exists ---
print(hasattr(curses.panel, 'panel'))          # True

# --- new_panel creates a panel ---
win1 = curses.newwin(5, 20, 0, 0)
pan1 = curses.panel.new_panel(win1)
print(pan1 is not None)                        # True

# --- window() returns the associated window ---
print(pan1.window() is win1)                   # True

# --- hidden() starts False ---
print(pan1.hidden() == False)                  # True

# --- hide() and show() ---
pan1.hide()
print(pan1.hidden() == True)                   # True
pan1.show()
print(pan1.hidden() == False)                  # True

# --- set_userptr / userptr ---
pan1.set_userptr('my_data')
print(pan1.userptr() == 'my_data')             # True

# set to arbitrary object
pan1.set_userptr(42)
print(pan1.userptr() == 42)                    # True

# --- replace(win) ---
win2 = curses.newwin(5, 20, 0, 0)
pan1.replace(win2)
print(pan1.window() is win2)                   # True

# --- move(y, x) ---
pan1.move(2, 5)
print(True)   # no exception                   # True

# --- update_panels is a no-op ---
curses.panel.update_panels()
print(True)                                    # True

# --- Stack ordering: new panels go on top ---
win3 = curses.newwin(5, 20, 0, 0)
win4 = curses.newwin(5, 20, 0, 0)
pan2 = curses.panel.new_panel(win3)
pan3 = curses.panel.new_panel(win4)
# pan3 is on top (most recently added)
top = curses.panel.top_panel()
print(top is pan3)                             # True

# --- bottom_panel ---
# pan1 was added first (of pan1/pan2/pan3), so it's at bottom
bottom = curses.panel.bottom_panel()
print(bottom is pan1)                          # True

# --- top() moves panel to top ---
pan1.top()
new_top = curses.panel.top_panel()
print(new_top is pan1)                         # True

# --- bottom() moves panel to bottom ---
pan1.bottom()
new_bottom = curses.panel.bottom_panel()
print(new_bottom is pan1)                      # True

# --- above() ---
# After pan1.bottom(): stack is [pan1, pan2, pan3]
# pan1.above() should be pan2
above_pan1 = pan1.above()
print(above_pan1 is pan2)                      # True

# --- below() ---
# pan2.below() should be pan1
below_pan2 = pan2.below()
print(below_pan2 is pan1)                      # True

# --- top panel is None when all hidden ---
win5 = curses.newwin(3, 10, 0, 0)
pan4 = curses.panel.new_panel(win5)
pan4.hide()
# top_panel returns last visible; if pan3 is visible it's the top
# (pan4 is on top of stack but hidden, so top_panel returns pan3)
tp2 = curses.panel.top_panel()
print(tp2 is not pan4)                         # True (pan4 is hidden)

# --- above() returns None at top ---
pan4.show()   # show pan4 again so it's the real top
pan4.top()    # ensure pan4 is at top
top_panel = curses.panel.top_panel()
print(top_panel is pan4)                       # True
above_top = pan4.above()
print(above_top is None)                       # True

# --- below() returns None at bottom ---
pan1.bottom()   # ensure pan1 is at bottom
below_bottom = pan1.below()
print(below_bottom is None)                    # True

# --- isinstance check ---
print(isinstance(pan1, curses.panel.panel))    # True
