# Afterburner instructions

1. Install Afterburner. It will also install RivaTuner statistics server.
2. Open Afterburner, go to Settings, then "Monitoring" tab.
3. Change "Hardware polling period (in miliseconds)" to "100"
4. Modify the graphs:
	1. Disable everything (tip: select first element, then SHIFT+select last element and click on one of the checkboxes in the list to enable/disable all)
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
5. Check "Log history to file" and select location for that file (e.g. Desktop or Downloads).
6. Check "Recreate existing log files"
7. Set "Begin logging" and "End logging" shortcuts. Suggestion is SHIFT+F2 and SHIFT+F3 appropriately.
8. Close settings.
9. Ensure that AfterBurner is opened and RivaTuner service is running in system tray.
10. Start the game, overlay will show up automatically in 5-30 seconds.
11. When starting benchmark, press shortcut to record, then press shortcut to stop recording. Note that there is no indication that game is being recorded or not.

You will end up with a file, named `TODO`. Rename it to a label that you want to see in the website. Something like `Windows` or `something else` (with or without `.hml` extension) will work.
