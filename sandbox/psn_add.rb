require 'prawn'
require 'rqrcode'
require 'chunky_png'
require 'cgi'
require 'net/http'
require 'uri'
require 'stringio'

data = "
Aaron Tishler
	GrenadeMagnet13

Adam Crawley
	Mollikye

Antonio Revard
	Project-Leet

Bruno Parrinello
	brunoinnerg

Christian Trinidad
	Novellito

Dane Christensen
	Imjustaseal

Daniel Luka
	cptballoonhands

Eleanor Wright
	Ebzdi

Emanuele Tozzato
	psychic-disco2

Jeffrey Yip
	Ethyx009

Mikkel Gadegaard
	AnkerHugz

Nelson Izquierdo
	Swish3r233

PJ Camp Malik
	oaceofspades

Robert Newlin
	Cpt_Newlin

William (Scott) Watson
	wswatson

Mark Muller
	Smash0r555

Kelly Fitzgerald
	CrashMcCrunchy17

Phil Sinnott
	pococrib

Justin Crawford
	nitsuj33

Mitch Snow
	MSnowjob

Pascal Maheux
	lascap_

Ben Frost
	Frozn3v0ltn

Shawn McCray
	ShawnyTwoWheels

Matt Richardson
	MattRich1888

Ryan Williams
	SoulCrush3r-RW

Emanuele Tozzato
	psychic-disco2

Shane Castello
	Nanablock209

Braxton Welch
	tobogganMantis0

Gabe Romero
	NearSighter-

Meng Cheng
	mengchch

Wes Limtiaco
	worldwide_wes

Chris Yong-Set
	PickSlid3

  Aiden Rimmer
	AidsR123

Andrew Wilson
	MonkeyWrenchGuy

Antonio Grasso
	antoskater86

Ash Clarke
	its-mr-ned

Daniel Pavel
	dixy15

Daniel Thavapalan
	DANIEL_JT

David Reyes
	VgerStorm

David Schumacher
	narren

Guillaume Herail
	xiu_42

Jack Stokes
	Jacktacular91

Jonathan Harris-French
	MisterBenson

Jonny McHale
	skimbatcha

Jordan Atkins
	sound_of_lies

Julian Huijbregts
	JulianX85

Marijn Rivière
	MarienTheRiver

Mark Hall
	MagnanimousKilla

Mattia Accinelli
	malavock69

Niall McGuinness
	milkslice

Oliver Collett
	olicollett

Dr. Panos Tessaromatis
	froggykrueger

Phil Valiente
	pil2409

Stephen Wall
	Milky_Magic

Tanya Wayman
	inuzuka85

Tom Learmouth
	MiniatureDJ

Tom Viljevac
	tomigun01ps

Willem Janssen
	Wlm1010010

Will Cox
	Kiwadoi

Daniele Antinolfi
	DanAntZ

Rahul Govind
	MidnightOwl92

Dan Parry
	pazza1665

Chris McCann
	macca1985

Rob Gandy
	Lister_TheStupid

Jay Santokhi
	Jay1999neo

Adam Fox
	Giberlator

Andrey Tretyakov
	a33point3

Katsuhiko Akita
	mit

Hirotaka Ishikawa
	chuko145

Nicholas Jong
	kenjishii
"
user_pairs = data.split("\n").reject(&:empty?).each_slice(2).to_a

def fetch_profile_page(profile_url)
  uri = URI(profile_url)

  Net::HTTP.start(
    uri.host,
    uri.port,
    use_ssl: uri.scheme == 'https',
    open_timeout: 3,
    read_timeout: 3
  ) do |http|
    response = http.get(uri.request_uri)
    response.body.to_s
  end
rescue StandardError
  nil
end

def profile_public?(profile_html)
  profile_html && !profile_html.include?('error_search.png') && !profile_html.include?('Something went wrong')
end

profile_cache = {}

Prawn::Document.generate("playstation_profiles.pdf") do
  # Add the font
  font_families.update(
    "Nerd" => {
      normal: "./TerminessNerdFont-Regular.ttf",
      bold:   "./TerminessNerdFont-Bold.ttf"
    }
  )

  # Set the page background color to dark
  canvas do
    fill_color '000000'  # Dark background
    fill_rectangle [bounds.left, bounds.top], bounds.right, bounds.top
  end

  font "Nerd"

  user_pairs.each do |name, username|
    puts "#{name.strip},#{username.strip}"

    online_id = username.strip
    profile_url = "https://profile.playstation.com/#{CGI.escape(online_id)}"
    profile_html = profile_cache[online_id] ||= fetch_profile_page(profile_url)
    is_public_profile = profile_public?(profile_html)

    # Check if there's enough space left on the page
    if cursor < (is_public_profile ? 260 : 220)
      start_new_page
      canvas do
        fill_color '000000'  # Dark background
        fill_rectangle [bounds.left, bounds.top], bounds.right, bounds.top
      end
    end

    # Group the content together
    bounding_box([0, cursor], width: bounds.width) do
      # Set color for name
      font("Nerd", style: :bold) do
        fill_color "9146FF"  # Dark Blue
        text "Name: #{name.strip}"
      end

      # Set color for username
      font("Nerd", style: :normal) do
        fill_color "c96ce8"  # Purple
        text "Online ID: #{online_id}"
      end

      # Set color for URL
      font("Nerd", style: :normal) do
        fill_color "4dbb71"  # Green
        text "Profile URL: #{profile_url}"
      end

      font("Nerd", style: :normal) do
        fill_color(is_public_profile ? '7CFC98' : 'FFB347')
        text "Status: #{is_public_profile ? 'Public profile' : 'Private or unavailable'}"
      end

      font("Nerd", style: :normal) do
        fill_color "f2f2f2"
        if is_public_profile
          text "Public profiles get the profile QR plus the Online ID QR."
        else
          text "Private profiles skip the broken profile QR and keep the Online ID QR as the fallback."
        end
      end

      online_id_qr_code = RQRCode::QRCode.new(online_id)
      online_id_png = online_id_qr_code.as_png(
        size: 256,
        border_modules: 2,

        color: 'black',
        fill: 'white'
      )

      move_down 5
      qr_top = cursor

      if is_public_profile
        profile_qr_code = RQRCode::QRCode.new(profile_url)
        profile_png = profile_qr_code.as_png(
          size: 256,
          border_modules: 2,

          color: 'black',
          fill: 'white'
        )

        bounding_box([0, qr_top], width: bounds.width / 2 - 10) do
          fill_color '4dbb71'
          text 'Profile page QR'
          move_down 5
          image StringIO.new(profile_png.to_s), width: 140, height: 140
        end

        bounding_box([bounds.width / 2 + 10, qr_top], width: bounds.width / 2 - 10) do
          fill_color 'f2f2f2'
          text 'Online ID QR'
          move_down 5
          image StringIO.new(online_id_png.to_s), width: 140, height: 140
        end
      else
        bounding_box([bounds.width / 2 - 75, qr_top], width: 150) do
          fill_color 'f2f2f2'
          text 'Online ID QR'
          move_down 5
          image StringIO.new(online_id_png.to_s), width: 150, height: 150
        end
      end

      unless is_public_profile
        move_down 20
        fill_color 'f2f2f2'
        text 'Private profile QR hidden because PlayStation returns an error page for this ID.'
      end

      # Add space between entries
      move_down(is_public_profile ? 170 : 190)
    end
  end
end