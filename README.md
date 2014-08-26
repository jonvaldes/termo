termo
=====

Super simple ncurses-style terminal drawing lib in Go.
Made this small lib as I was having trouble with termbox when updating the whole screen.

You can check out an example program here: https://github.com/jonvaldes/termo_example

Only tested in OSX for a couple of simple projects, so I wouldn't advise relying on this for serious stuff.


Known problem
-------------

When exiting the app, the terminal might be left in a weird state where it can't autocomplete correctly. Not sure yet why this happens, though.  
