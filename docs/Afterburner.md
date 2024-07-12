# Afterburner instructions

1. Install Afterburner. It will also install RivaTuner statistics server.
2. Open Afterburner, go to Settings, then "Monitoring" tab.
3. Change "Hardware polling period (in miliseconds)" to "100" (ensure ALL your benchmarks have the same interval value, even on Linux!)
4. Modify the graphs:
	1. Disable everything
	2. Enable the following:
		* GPU temperature
		* GPU usage
		* Memory usage
		* Core clock
		* Memory clock
		* Power
		* CPU temperature
		* CPU usage
		* RAM usage
		* Framerate
		* Frametime
	3. (optional) Click on each, then check "Show in On-Screen Display" for each, so you can see them in Overlay
5. Check "Log history to file"
6. Select location for such file (e.g. Desktop or Downloads works great).
7. Check "Recreate existing log files"
8. Uncheck "Log history to file" (yes, check to configure and then uncheck to disable _auto_ recording when game starts)
9. Set "Begin logging" and "End logging" shortcuts. Suggestion is SHIFT+F2 and SHIFT+F3 appropriately.
10. Close Afterburner settings.
11. Ensure that AfterBurner and RivaTuner is running (opened or in system tray).
12. Start the game, overlay will show up in 5-30 seconds (keep clicking a mouse when the game is loading)
13. When starting benchmark, press shortcut to record, then press shortcut to stop recording. Note that there is no indication that game is being recorded or not.

You will end up with a file, named `*.hml`. Rename it to a label that you want to see in the website. Something like `Windows` or `something else` (with or without `.hml` extension) will work.
