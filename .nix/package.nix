{
	src ? ./..,
	lib,
	pkgs,
	internalPkgs ? import ./pkgs.nix {}, # only for overriding
}:

let desktopFile = pkgs.makeDesktopItem {
    desktopName = "Application Dash";
	name = "GAppDash";
	icon = "gappdash";
	exec = "gappdash";
};

in internalPkgs.buildGoModule {
	inherit src;

	pname = "gappdash";
	version = "0.0.1-tip";

	# Bump this on go.mod change.
	# vendorSha256 = "1v8mlawbl011696xlw839s9j956pyygpff924v1zbq3bpfylxqp4";

	buildInputs = with internalPkgs; [
		gnome.gtk3
		glib
		gdk-pixbuf
		gobjectIntrospection
	];

	nativeBuildInputs = with pkgs; [ pkgconfig ];

	preFixup = ''
		mkdir -p $out/share/icons/hicolor/256x256/apps/ $out/share/applications/
		# Install the desktop file
		cp "${desktopFile}"/share/applications/* $out/share/applications/
		# Install the icon
		cp "${../.github/logo-256.png}" $out/share/icons/hicolor/256x256/apps/gotktrix.png
	'';
}
