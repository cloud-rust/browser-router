# Linux desktop-less test machine: Vivaldi and Firefox under Xvfb, xdg-utils for the default browser.
FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates gnupg xvfb xauth sqlite3 procps \
      xdg-utils dbus-x11 python3 \
 && wget -qO- https://repo.vivaldi.com/archive/linux_signing_key.pub | gpg --dearmor > /usr/share/keyrings/vivaldi.gpg \
 && echo "deb [signed-by=/usr/share/keyrings/vivaldi.gpg] https://repo.vivaldi.com/archive/deb/ stable main" > /etc/apt/sources.list.d/vivaldi.list \
 && install -d -m 0755 /etc/apt/keyrings \
 && wget -qO /etc/apt/keyrings/packages.mozilla.org.asc https://packages.mozilla.org/apt/repo-signing-key.gpg \
 && echo "deb [signed-by=/etc/apt/keyrings/packages.mozilla.org.asc] https://packages.mozilla.org/apt mozilla main" > /etc/apt/sources.list.d/mozilla.list \
 && printf 'Package: *\nPin: origin packages.mozilla.org\nPin-Priority: 1000\n' > /etc/apt/preferences.d/mozilla \
 && apt-get update && apt-get install -y --no-install-recommends vivaldi-stable firefox \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m tester
USER tester
